"""Emits the provider's API usage mapping against the OAS browser contract."""
import argparse, json, collections, os, re, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import resolve as r

ap = argparse.ArgumentParser(description=__doc__)
ap.add_argument("sites", help="output of the callsites tool")
ap.add_argument("artifacts", help="output of the callgraph tool")
ap.add_argument("-o", "--out", default="api-usage-mapping.json")
ap.add_argument("--corpus", default=None,
                help="a checkout of the powerplatform-apis repository")
ap.add_argument("--catalogue",
                default="https://adamcoulteroz.github.io/powerplatform-apis/specs.json")
args = ap.parse_args()

r.load_corpus(args.corpus)

sites = {f"{s['file']}:{s['line']}": s for s in json.load(open(args.sites))}
arts = json.load(open(args.artifacts))

UNCATALOGUED = {
    'async-continuation': 'Follows a Location or Operation-Location header returned by a previous response.',
    'caller-supplied':    'The request URL is supplied by the practitioner at run time.',
}

def uncatalogued_reason(s):
    if s.get('dynamic', '').startswith('follows') or 'location' in (s.get('url_expr') or '').lower():
        return 'async-continuation'
    return 'caller-supplied'

IDENT = re.compile(r'^[A-Za-z_][A-Za-z0-9_]*$')

def tidy_path(p):
    """Parameter names are arbitrary under the matching rule, so render any
    placeholder that is a Go expression rather than a name as {value}."""
    return re.sub(r'\{([^{}]*)\}',
                  lambda m: '{' + (m.group(1) if IDENT.match(m.group(1)) else 'value') + '}',
                  p)

def readable(fn):
    """environment.(Client).updateEnvironmentAiFeaturesWithRetry -> updateEnvironmentAiFeatures..."""
    return re.sub(r'^[a-z_]+\.(\([^)]*\)\.)?', '', fn)

# (item, entrypoint, spec, operation) -> row
rows = collections.OrderedDict()
uncat = collections.defaultdict(collections.Counter)
version_use = collections.defaultdict(set)

def item_key(a):
    # a resource and a data source may share a Terraform address, so identity
    # is the pair; the address alone is the display name.
    return f"{a['kind']}:{a['address']}"

for a in arts:
    key = item_key(a)
    for o in a['operations']:
        s = sites.get(o['site'])
        if not s:
            continue
        paths = s.get('path_variants') or [s.get('path')]
        if not s.get('path') or s['http_method'].startswith('$'):
            uncat[key][uncatalogued_reason(s)] += 1
            continue
        for p in paths:
            sp, oid, note = r.resolve(s['boundary'], s['http_method'], p)
            if not oid:
                # names an operation the catalogue does not carry; emit it as the
                # code spells it so the failure stays visible
                sp = sp or s['boundary']
                oid = None
            k = (key, o['entrypoint'], sp, oid or f"{s['http_method'].lower()} {p}")
            row = rows.get(k)
            if row is None:
                row = {'spec': sp, 'entrypoint': o['entrypoint'],
                       'conditional': o['conditional'], 'approximate': bool(s.get('approximate')),
                       'via': readable(o['call_path'][-1]), 'apiVersion': s.get('api_version') or None}
                if oid:
                    row['operation'] = oid
                else:
                    row['method'] = s['http_method'].lower()
                    row['path'] = tidy_path(p)
                rows[k] = row
            else:
                # any unconditional route wins
                row['conditional'] = row['conditional'] and o['conditional']
                row['approximate'] = row['approximate'] and bool(s.get('approximate'))
            if oid and s.get('api_version'):
                version_use[(sp, oid)].add(s['api_version'])

items = collections.OrderedDict()
for a in arts:
    items.setdefault(item_key(a), {
        'id': item_key(a), 'kind': a['kind'], 'name': a['address'],
        'source': {'path': a['source']['path'], 'line': a['source']['line']},
        'calls': [],
    })

for (key, ep, sp, oid), row in rows.items():
    call = {'spec': row['spec']}
    if 'operation' in row:
        call['operation'] = row['operation']
    else:
        call['method'] = row['method']; call['path'] = row['path']
    call['entrypoint'] = ep
    call['coverage'] = 'partial' if row['conditional'] else 'full'
    call['grade'] = 'derived'
    # a version is an annotation only where this caller pins one of several
    if row['apiVersion'] and len(version_use.get((sp, oid), ())) > 1:
        call['apiVersion'] = row['apiVersion']
    if row['conditional']:
        call['note'] = f"Reached only through {row['via']}."
    if row['approximate']:
        call['approximate'] = True
        call['note'] = (call.get('note', '') + ' Path assembled by conditional reassignment; not resolved to a branch.').strip()
    items[key]['calls'].append(call)

for key, counts in uncat.items():
    items[key]['uncatalogued'] = [
        {'reason': reason, 'count': n, 'note': UNCATALOGUED[reason]}
        for reason, n in sorted(counts.items())
    ]

# One path position must be spelled one way across every row, or two rows for
# one operation read as two operations.
shapes = collections.defaultdict(list)
for i in items.values():
    for c in i['calls']:
        if 'path' in c:
            shapes[(c['method'], re.sub(r'\{[^{}]*\}', '{}', c['path']))].append(c)

for (method, shape), calls in shapes.items():
    spellings = [re.findall(r'\{([^{}]*)\}', c['path']) for c in calls]
    canonical = []
    for pos in range(len(spellings[0])):
        named = [sp[pos] for sp in spellings if sp[pos] != 'value']
        canonical.append(named[0] if named else 'value')
    it = iter(canonical)
    path = re.sub(r'\{[^{}]*\}', lambda _: '{' + next(it) + '}', calls[0]['path'])
    for c in calls:
        c['path'] = path

doc = {
    'catalogue': args.catalogue,
    'artifacts': {
        'kind': 'provider component',
        'kinds': {'resource': 'Resource', 'datasource': 'Data source'},
        'entrypoints': {'Create': 'Create', 'Read': 'Read', 'Update': 'Update',
                        'Delete': 'Delete', 'ImportState': 'Import',
                        'ModifyPlan': 'Plan modification', 'ValidateConfig': 'Config validation'},
        'uncatalogued': {'async-continuation': 'Async continuation',
                         'caller-supplied': 'Caller-supplied URL'},
    },
    'grades': {
        'observed': 'observed',
        'vocabulary': [
            {'id': 'observed', 'title': 'Observed in recorded traffic',
             'caveat': 'Seen on the wire in a recorded exchange.'},
            {'id': 'derived', 'title': 'Derived from source',
             'caveat': 'Read out of the provider source by static analysis, not seen executing.'},
        ],
    },
    'items': sorted(items.values(), key=lambda i: i['id']),
}

json.dump(doc, open(args.out, 'w'), indent=2)

n_calls = sum(len(i['calls']) for i in doc['items'])
n_uncat = sum(len(i.get('uncatalogued', [])) for i in doc['items'])
n_unres = sum(1 for i in doc['items'] for c in i['calls'] if 'operation' not in c)
n_part = sum(1 for i in doc['items'] for c in i['calls'] if c['coverage'] == 'partial')
n_ver = sum(1 for i in doc['items'] for c in i['calls'] if 'apiVersion' in c)
n_appr = sum(1 for i in doc['items'] for c in i['calls'] if c.get('approximate'))
print(f"items            {len(doc['items'])}")
print(f"calls            {n_calls}   (partial {n_part}, full {n_calls-n_part})")
print(f"  no operationId {n_unres}")
print(f"  apiVersion set {n_ver}")
print(f"  approximate    {n_appr}")
print(f"uncatalogued entries {n_uncat}")
