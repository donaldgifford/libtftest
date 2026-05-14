// Package services hosts the per-service handlers for sneakystack's
// LocalStack-gap-filling proxy.
//
// Each handler implements the AWS protocol it serves
// (JSON-RPC for AWS json-1.1 APIs like SSO Admin and Organizations,
// REST-XML for older services), reads and mutates state via the
// sneakystack [sneakystack.Store] interface, and is registered with
// the sneakystack [sneakystack.Server] at startup.
//
// New gap-fillers go here. Use the libtftest:add-sneakystack-service
// skill to scaffold the JSON-RPC or REST-XML template.
package services
