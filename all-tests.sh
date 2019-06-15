#!/usr/bin/env bash

fail=0

for test_script in `ls *-tests.sh`
do
    if [ $test_script != "all-tests.sh" ]; then
        echo >&2 "RUNNING $test_script"
        ./$test_script
        if [[ $? != 0 ]]; then
            echo ${test_script//.sh/} failed.
            fail=1
        fi
    fi
done

if [[ $fail == 0 ]]; then
    echo >&2 "All tests passed."
fi

exit $fail
