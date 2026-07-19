// Package modules groups domain ownership for the modular monolith.
//
// Domain ownership (keep one Postgres; do not split DBs yet):
//
//   - auth:    login, JWT, 2FA, users, password flows
//   - crm:     companies, customers, solutions, PO assignments, dashboards
//   - tickets: tickets, feedback, uploads, SLA (worker)
//   - amc:     AMC assignments/visits/proofs, contract expiry views (worker)
//   - notify:  in-app notifications and preferences
//
// HTTP route registration lives in each subpackage; wiring stays in
// internal/bootstrap so the API remains a single binary until a real
// scale/team need forces a service extract.
package modules
