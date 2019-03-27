#! /bin/bash

set -o pipefail

export RESULT_XML_FILE_NAME="/tmp/test-reports/result.xml"
DIR="$(dirname $0)"
ESC_SEQ="\033["
CONSOLE_RESET=$ESC_SEQ"39;49;00m"
COLOR_RED=$ESC_SEQ"31;01m"
COLOR_GREEN=$ESC_SEQ"32;01m"
COLOR_YELLOW=$ESC_SEQ"33;01m"
COLOR_CYAN=$ESC_SEQ"36;01m"
CONSOLE_BOLD=$ESC_SEQ"1m"

function init_result_xml {
    mkdir /tmp/test-reports
    rm -f $RESULT_XML_FILE_NAME
    text_count=$(find $DIR/../test_cases/ -mindepth 1 -maxdepth 1 -type d | wc -l)
    echo '<?xml version="1.0" encoding="UTF-8"?>' >> $RESULT_XML_FILE_NAME
    echo '<testsuite name="Kore end to end tests"' >> $RESULT_XML_FILE_NAME
    echo "    tests=\"$test_count\"" >> $RESULT_XML_FILE_NAME
    echo '    id="0"' >> $RESULT_XML_FILE_NAME
    echo '    package="test">' >> $RESULT_XML_FILE_NAME
    echo '' >> $RESULT_XML_FILE_NAME
}

function init_test_xml {
    echo "<testcase name=\"$1\"" >> $RESULT_XML_FILE_NAME
    echo 'classname="EndToEnd">' >> $RESULT_XML_FILE_NAME
}

function finish_test_xml {
    if [ $1 != 0 ]; then
        echo '<failure message="test failed"><![CDATA[' >> $RESULT_XML_FILE_NAME
        echo "$(cat $2)" >> $RESULT_XML_FILE_NAME
        echo ']]></failure>' >> $RESULT_XML_FILE_NAME
    fi
    output=$(cat $2)
    echo "<system-out><![CDATA[" >> $RESULT_XML_FILE_NAME
    echo "$output" >> $RESULT_XML_FILE_NAME
    echo "]]></system-out>" >> $RESULT_XML_FILE_NAME
    echo "</testcase>" >> $RESULT_XML_FILE_NAME
}

function finish_result_xml {
    echo "</testsuite>" >> $RESULT_XML_FILE_NAME
}

init_result_xml

success=1
for test_dir in `find $DIR/ -mindepth 1 -maxdepth 1 -type d`
do
    test_name=$(basename $test_dir)
    out_file=$(mktemp)

    init_test_xml $test_name
    echo -e "${CONSOLE_BOLD}${COLOR_GREEN}--------------------------------------------------------------------------------------${CONSOLE_RESET}"
    echo -e "${CONSOLE_BOLD}${COLOR_GREEN}--- Running: $test_dir${CONSOLE_RESET}"
    ./$test_dir/run.sh |& tee $out_file
    exit_code=$?
    finish_test_xml $exit_code $out_file
    echo -e "${CONSOLE_BOLD}${COLOR_GREEN}--- Done: $testdir${CONSOLE_RESET}"
    if [ $exit_code == 0 ]; then
        echo -e "${CONSOLE_BOLD}${COLOR_GREEN}--- Test was Successfull. ${CONSOLE_RESET}"
    else
        echo -e "${CONSOLE_BOLD}${COLOR_RED}--- Test failed. ${CONSOLE_RESET}"
        success=0
    fi
    echo -e "${CONSOLE_BOLD}${COLOR_GREEN}--------------------------------------------------------------------------------------${CONSOLE_RESET}"
done

finish_result_xml

if [ $success == 0 ]; then
    exit 1
fi
