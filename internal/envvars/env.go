package envvars

import "os"

// ForceVerify indicates that all verifications should be performed, even if
// e.g. SkipBuggyChecks() says otherwise.
var ForceVerify bool = os.Getenv("THEMA_FORCEVERIFY") == "1"
