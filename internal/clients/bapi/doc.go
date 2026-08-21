// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package bapi is the client library for the Business Application Platform
// API (BAPI), the undocumented control-plane API behind the Power Platform
// admin center (public cloud host: api.bap.microsoft.com).
//
// The package owns the BAPI wire shapes (DTOs) and request logic, including
// single-boundary workflows such as lifecycle-operation polling and conflict
// retry loops. Workflows that span multiple API boundaries belong to the
// service layer in internal/services.
//
// Requests authenticate with the PowerApps service scope
// (https://service.powerapps.com/.default in the public cloud).
package bapi
