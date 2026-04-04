package eval

import "context"

// Subject is a function that executes the system under test and populates
// an [Observation]'s Output, Annotations, and Timing fields.
//
// The framework provides Input and GroundTruth from the dataset.
// The Subject is responsible for:
//   - Unmarshaling Input into the appropriate type
//   - Calling the system under test
//   - Serializing the result into Output
//   - Attaching subsystem-specific data to Annotations under well-known keys
//   - Recording timing information
type Subject func(ctx context.Context, obs *Observation) error
