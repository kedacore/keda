#!/bin/sh

export JAVA_HOME=/root/buildbox/jdk1.8.0_151
export CODEDEX_PATH=/root/buildbox/CodeDEX_V3
export CODEMARS_HOME=$CODEDEX_PATH/tool/tools/codemars_Newest
export GO_HOME=/root/buildbox/go-1.8.1

#path
export PATH=$GO_HOME/bin:$CODEMARS_HOME/bin:$JAVA_HOME/bin:$PATH

# FORTIFY_BUILD_ID可设置自己服务的ID值，这个ID名由数字、字母、下划线组成，不支持包含中划线-
export FORTIFY_BUILD_GO=fuxi_codedex_test1

#使用环境变量INTER_DIR(中间文件目录)、SRC_WS(源码路径)、SCAN_DIR(待扫描的代码相对路径)
export inter_dir=$INTER_DIR
export codemars_tmp_dir=$inter_dir/codemars_tmp
export project_root=$SRC_WS/$SCAN_DIR

rm -rf $inter_dir

# codemars

cd $CODEMARS_HOME
sh CodeMars.sh -go -source $project_root -output $codemars_tmp_dir/CodeMars.json

cd $codemars_tmp_dir
zip codemars.zip CodeMars.json
cp codemars.zip $inter_dir/