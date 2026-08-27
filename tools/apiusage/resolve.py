"""Resolves a method and path against the OpenAPI corpus.

The corpus is a checkout of the powerplatform-apis repository. Point
POWERPLATFORM_APIS at it, or pass its path to load_corpus().
"""
import json, glob, os, re, collections

CORPUS = os.environ.get("POWERPLATFORM_APIS",
                        os.path.expanduser("~/Code/GitHub/AdamCoulterOz/powerplatform-apis"))

def segs(p):
    p=p or ''
    if not p.startswith('/'): p='/'+p
    return p.rstrip('/').split('/')[1:]

def ph(seg):  # whole segment is a single template parameter
    return seg.startswith('{') and seg.endswith('}') and seg.count('{')==1

def normseg(seg):
    return re.sub(r'\{[^{}]*\}','{}',seg)

def segmatch(spec, call):
    """A templated part of a spec segment absorbs whatever the call has there.

    The whole-segment case is the common one, but a spec also templates inside a
    segment, as OData does with {entitySetName}({recordId}). Everything outside
    a template must match exactly, case included.
    """
    if ph(spec):
        return True
    pattern = "".join(
        "[^/]+" if part.startswith("{") and part.endswith("}") else re.escape(part)
        for part in re.split(r"(\{[^{}]*\})", spec) if part)
    return re.fullmatch(pattern, call) is not None


def match(specSegs, callSegs):
    if len(specSegs)!=len(callSegs): return False
    return all(segmatch(a,b) for a,b in zip(specSegs,callSegs))

SPECS = {}


def load_corpus(root=None):
    """Indexes every operation in every spec under root."""
    root = root or CORPUS
    found = sorted(glob.glob(os.path.join(root, "*", "oas", "openapi.json")))
    if not found:
        raise SystemExit(
            f"no specs under {root}. Point POWERPLATFORM_APIS at a checkout of "
            "the powerplatform-apis repository.")
    SPECS.clear()
    for p in found:
        n = p.split(os.sep)[-3]
        spec = json.load(open(p))
        ops = []
        for path, item in spec.get("paths", {}).items():
            for m, op in item.items():
                if m.lower() not in ("get", "put", "post", "delete", "patch"):
                    continue
                ops.append({"method": m.upper(), "path": path,
                            "segs": segs(path), "id": op["operationId"]})
        SPECS[n] = ops
    return SPECS

B2S={'bapi':'bapi','dataverse':'dataverse','ppapi':'ppapi','ppapi-environment':'ppapi',
     'ppapi-tenant':'ppapi','admin':'admin','advisor':'advisor','copilot':'copilot',
     'analytics':'analytics','powerapps':'powerapps','licensing':'licensing'}

def specificity(o):
    # a candidate that matches on more literal segments is the better route
    return sum(0 if ph(x) else 1 for x in o['segs'])

def resolve(boundary, method, path):
    """Returns (spec, operationId, note) or (spec, None, reason)."""
    sp=B2S.get(boundary)
    if not sp: return (None,None,'boundary has no spec')
    cs=segs(path)
    hits=[o for o in SPECS[sp] if o['method']==method and match(o['segs'],cs)]
    if hits:
        best=max(specificity(o) for o in hits)
        top=[o for o in hits if specificity(o)==best]
        if len(top)==1: return (sp,top[0]['id'],None)
        return (sp,top[0]['id'],'ambiguous: '+','.join(h['id'] for h in top))
    ci=[o for o in SPECS[sp] if o['method']==method and match([s.lower() for s in o['segs']],[c.lower() for c in cs])]
    if ci: return (sp,None,'case-only match with '+ci[0]['path'])
    return (sp,None,'absent')
