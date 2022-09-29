package envvars

import "os"

// ForceVerify indicates that all verifications should be performed, even if
// e.g. SkipBuggyChecks() says otherwise.
var ForceVerify = os.Getenv("THEMA_FORCEVERIFY") == "1"

// ReverseTranslate indicates whether reverse translation is supported.
//
// Used primarily as a single point of control for testing. Will be set
// permanently to true once support is finalized.
var ReverseTranslate = os.Getenv("THEMA_REVERSETRANSLATE") == "1"
