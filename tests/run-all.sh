#! /bin/bash
set -eu

E2E_REGEX=${E2E_TEST_REGEX:-*.test.ts}

DIR=$(dirname "$0")
cd $DIR

concurrent_tests_limit=6
pids=()
lookup=()
failed_count=0
failed_lookup=()
counter=0

function run_setup {
    ./node_modules/.bin/ava setup.test.ts
}

function run_tests {
    counter=0
    # randomize tests order using shuf
    for test_case in $(find scalers -name "$E2E_REGEX" | shuf)
    do
        if [[ $test_case != *.test.ts ]] # Skip helper files
        then
            continue
        fi

        counter=$((counter+1))
        ./node_modules/.bin/ava $test_case > "${test_case}.1.log" 2>&1 &
        pid=$!
        echo "Running $test_case with pid: $pid"
        pids+=($pid)
        lookup[$pid]=$test_case
        # limit concurrent runs
        if [[ "$counter" -ge "$concurrent_tests_limit" ]]; then
            wait_for_jobs
            counter=0
            pids=()
        fi
    done

    wait_for_jobs

    # Retry failing tests
    if [ ${#failed_lookup[@]} -ne 0 ]; then

        printf "\n\n##############################################\n"
        printf "##############################################\n\n"
        printf "FINISHED FIRST EXECUTION, RETRYING FAILING TESTS"
        printf "\n\n##############################################\n"
        printf "##############################################\n\n"

        retry_lookup=("${failed_lookup[@]}")
        counter=0
        pids=()
        failed_count=0
        failed_lookup=()

        for test_case in "${retry_lookup[@]}"
        do
            counter=$((counter+1))
            ./node_modules/.bin/ava $test_case > "${test_case}.2.log" 2>&1 &
            pid=$!
            echo "Rerunning $test_case with pid: $pid"
            pids+=($pid)
            lookup[$pid]=$test_case
            # limit concurrent runs
            if [[ "$counter" -ge "$concurrent_tests_limit" ]]; then
                wait_for_jobs
                counter=0
                pids=()
            fi
        done
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
    done

    printf "\n$failed_count jobs failed\n"
    printf '%s\n' "${failed_lookup[@]}"
}

function print_logs {
    for test_log in $(find scalers -name "*.log")
    do
        echo ">>> $test_log <<<"
        cat $test_log
        printf "\n\n##############################################\n"
        printf "##############################################\n\n"
    done

    echo ">>> KEDA Operator log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-operator | xargs kubectl -n keda logs
    printf "\n\n##############################################\n"
    printf "##############################################\n\n"

    echo ">>> KEDA Metrics Server log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-metrics-apiserver | xargs kubectl -n keda logs
    printf "\n\n##############################################\n"
    printf "##############################################\n\n"
}

function run_cleanup {
    ./node_modules/.bin/ava cleanup.test.ts
}

function print_failed {
    echo "$failed_count e2e tests failed"
    for failed_test in "${failed_lookup[@]}"; do
        echo $failed_test
    done
}

run_setup
run_tests
wait_for_jobs
print_logs
run_cleanup

if [ "$failed_count" == "0" ];
then
    exit 0
else
    print_failed
    exit 1
fi
