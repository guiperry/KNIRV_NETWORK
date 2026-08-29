package schema

import sharedschema "knirvhasher/pkg/hashing/schema"

// TrainingFrame is the shared schema definition used by all pipeline stages.
// The canonical struct lives in knirvhasher/pkg/hashing/schema to prevent
// drift between independently-maintained copies.
type TrainingFrame = sharedschema.TrainingFrame
