#! /bin/bash
set -u

E2E_REGEX="./*${E2E_TEST_REGEX:-*_test.go}"

DIR=$(dirname "$0")
cd $DIR

concurrent_tests_limit=12
pids=()
lookup=()
failed_count=0
failed_lookup=()
counter=0
executed_count=0

function run_setup {
    go test -v -tags e2e -timeout 15m utils/setup_test.go
}

function excute_test {
    if [[ $1 != *_test.go ]] # Skip helper files
    then
        return
    fi
    counter=$((counter+1))
    go test -v -tags e2e -timeout 20m $1 > "$1.$2.log" 2>&1 &
    pid=$!
    echo "Running $1 with pid: $pid"
    pids+=($pid)
    lookup[$pid]=$1
    # limit concurrent runs
    if [[ "$counter" -ge "$concurrent_tests_limit" ]]; then
        wait_for_jobs
        counter=0
        pids=()
    fi
}

function run_chaos {
    counter=0
    test_case="chaos/chaos_test.go"
    test_log="chaos/chaos_test.go.1.log"

    # execute the test only if the regex includes it
    if [[ "$test_case" =~ $E2E_REGEX ]]; then
        printf "#################################################\n"
        printf "#################################################\n"
        printf "===============STARTING CHAOS TEST===============\n"
        printf "#################################################\n"
        printf "#################################################\n"
        excute_test $test_case 1
        wait_for_jobs

        echo ">>> $test_log <<<"
        cat $test_log
        printf "#################################################\n"
        printf "#################################################\n"
    fi
}

function run_tests {
    counter=0
    # randomize tests order using shuf
    for test_case in $(find . -not -path '*/utils/*' -not -path '*/chaos/*' -wholename "$E2E_REGEX" | shuf)
    do
        excute_test $test_case 1
    done

    wait_for_jobs

    # Retry failing tests
    if [ ${#failed_lookup[@]} -ne 0 ]; then

        printf "#################################################\n"
        printf "#################################################\n"
        printf "FINISHED FIRST EXECUTION, RETRYING FAILING TESTS\n"
        printf "#################################################\n"
        printf "#################################################\n"

        retry_lookup=("${failed_lookup[@]}")
        counter=0
        pids=()
        failed_count=0
        failed_lookup=()

        for test_case in "${retry_lookup[@]}"
        do
            excute_test $test_case 2
        done

        wait_for_jobs
    fi

    # Retry failing tests
    if [ ${#failed_lookup[@]} -ne 0 ]; then

        printf "#################################################\n"
        printf "#################################################\n"
        printf "FINISHED SECOND EXECUTION, RETRYING FAILING TESTS\n"
        printf "#################################################\n"
        printf "#################################################\n"

        retry_lookup=("${failed_lookup[@]}")
        counter=0
        pids=()
        failed_count=0
        failed_lookup=()

        for test_case in "${retry_lookup[@]}"
        do
            excute_test $test_case 3
        done

        wait_for_jobs
    fi
}

function mark_failed {
    failed_lookup[$1]=${lookup[$1]}
    let "failed_count+=1"
}

function wait_for_jobs {
    for job in "${pids[@]}"; do
        wait $job || mark_failed $job
        echo "Job $job finished"
        executed_count=$((executed_count+1))
    done

    printf "\n$failed_count jobs failed\n"
    printf '%s\n' "${failed_lookup[@]}"
}

function print_logs {
    for test_log in $(find . -name "*.log")
    do
        echo ">>> $test_log <<<"
        cat $test_log
        printf "##############################################\n"
        printf "##############################################\n"
    done

    echo ">>> KEDA Operator log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-operator | xargs kubectl -n keda logs
    printf "##############################################\n"
    printf "##############################################\n"

    echo ">>> KEDA Metrics Server log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-metrics-apiserver | xargs kubectl -n keda logs
    printf "##############################################\n"
    printf "##############################################\n"

    echo ">>> KEDA Admission Webhooks log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-admission| xargs kubectl -n keda logs
    printf "##############################################\n"
    printf "##############################################\n"
}

function print_chaos_logs {
    if [[ "chaos/chaos_test.go" =~ $E2E_REGEX ]]; then
        echo ">>> KEDA Operator log <<<"
        for test_log in $(find . -path '*/chaos/*' -name "*operator*.log")
        do
            printf "##############################################\n"
            printf "##############################################\n"
            echo ">>> $test_log <<<"
            printf "##############################################\n"
            cat $test_log
        done

        echo ">>> KEDA Metrics Server log <<<"
        for test_log in $(find . -path '*/chaos/*' -name "*ms*.log")
        do
            printf "##############################################\n"
            printf "##############################################\n"
            echo ">>> $test_log <<<"
            printf "##############################################\n"
            cat $test_log
        done
    fi
}

function run_cleanup {
    go test -v -tags e2e utils/cleanup_test.go
}

function print_failed {
    echo "$failed_count e2e tests failed"
    for failed_test in "${failed_lookup[@]}"; do
        echo $failed_test
    done
}

run_setup
run_tests
print_logs
run_chaos
print_chaos_logs
run_cleanup

if [ "$executed_count" == "0" ];
then
    echo "No test has been executed, please review your regex: '$E2E_TEST_REGEX'"
    exit 1
elif [ "$failed_count" != "0" ];
then
    print_failed
    exit 1
else
    exit 0
fi
