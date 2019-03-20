#!/usr/bin/env bash

echo >&2 "RUNNING flag-tests.sh:"
./flag-tests.sh
FLAG_RC=$?

echo >&2 "RUNNING linter-tests.sh:"
./linter-tests.sh
LINTER_RC=$?

echo >&2 "RUNNING eval-tests.sh:"
./eval-tests.sh
EVAL_RC=$?

if [[ $FLAG_RC != 0 || $LINTER_RC != 0 || $EVAL_RC != 0 ]]; then
    echo >&2 "Failed tests: \$FLAG_RC=$FLAG_RC \$LINTER_RC=$LINTER_RC \$EVAL_RC=$EVAL_RC"
    exit 1
fi

echo >&2 "All three tests passed."

exit 0
