#! /bin/bash
set -u

DIR=$(dirname "$0")
cd $DIR

export E2E_INSTALL_KAFKA="false"

# use only e2e test which support running on ARM and s390x
test_files=(
    "scalers/kubernetes_workload/kubernetes_workload_test.go"
    "scalers/cron/cron_test.go"
)

concurrent_tests_limit=5
pids=()
lookup=()
failed_count=0
failed_lookup=()
counter=0

function run_setup {
    go test -v -tags e2e utils/setup_test.go
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

function run_tests {
    counter=0

    for test_case in ${test_files[@]}
    do
        excute_test $test_case 1
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
            excute_test $test_case 2
        done

        wait_for_jobs
    fi

    # Retry failing tests
    if [ ${#failed_lookup[@]} -ne 0 ]; then

        printf "\n\n##############################################\n"
        printf "##############################################\n\n"
        printf "FINISHED SECOND EXECUTION, RETRYING FAILING TESTS"
        printf "\n\n##############################################\n"
        printf "##############################################\n\n"

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
    done

    printf "\n$failed_count jobs failed\n"
    printf '%s\n' "${failed_lookup[@]}"
}

function print_logs {
    for test_log in $(find . -name "*.log")
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

    echo ">>> KEDA Admission Webhooks log <<<"
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-admission | xargs kubectl -n keda logs
    printf "##############################################\n"
    printf "##############################################\n"
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
run_cleanup

if [ "$failed_count" == "0" ];
then
    exit 0
else
    print_failed
    exit 1
fi
