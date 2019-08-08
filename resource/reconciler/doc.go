// Package reconciler reconciles resources in a graph.
//
// Steps
//
// The reconciliation process consists at a high level of 3 steps:
//
//   1. Get existing resources
//
//      Existing resources for the project in the current namespace are listed.
//      Any previously created resources are returned.
//
//   2. Create or Update resources
//
//      The desired graph is walked in the order specified by the dependencies.
//      Each resource encountered is compared to existing resources.
//
//        - When an exact match is found, the walk continues to the next resource, with
//          the output passed to it.
//
//        - If an existing resource with the same type and name exists but the input
//          values do not match, the resource is updated.
//
//        - When no resource exists for the type-name combination, it is created.
//
//   3. Delete resources
//
//      Resources that were not matched in the create/update phase are cleaned up.
//      Thus, resources are always created in the desired state, before
//      anything gets removed.
//
// Concurrency
//
// When possible, changes are performed concurrently.
//
//   A and B execute concurrently.
//   When both have been executed (without error), C is executed.
//
//       A --> C
//       B -/
//
//   A is executed, then B & C concurrently, then D.
//
//       A -> B -> D
//         \- C -/
//
// Retries
//
// All operations are retried with exponential backoff. If a non-retryiable
// error occurs, the resource definition should wrap the returned error with
// backoff.PermanentError(err).
package reconciler
