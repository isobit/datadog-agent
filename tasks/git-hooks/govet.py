#!/usr/bin/env python3

import os.path
import subprocess
import sys

# Exclude some folders since go vet fails there
EXCLUDED_FOLDERS = {
    "./cmd/agent/android",
    "./cmd/agent/windows/service",
    "./cmd/cluster-agent",
    "./cmd/cluster-agent/app",
    "./cmd/systray",
    "./pkg/clusteragent/orchestrator",
    "./pkg/process/config/testdata",
    "./pkg/process/util/orchestrator",
    "./pkg/trace/test/testsuite/testdata",
    "./pkg/util/containerd",
    "./pkg/util/containers/cri/crimock",
    "./pkg/util/containers/providers/cgroup",
    "./pkg/util/containers/providers/windows",
    "./pkg/util/hostname/apiserver",
    "./pkg/util/log",
    "./pkg/util/log/zap",
    "./pkg/util/scrubber",
    "./pkg/obfuscate",
    "./pkg/obfuscate/testdata",
    "./pkg/otlp/model",
    "./pkg/otlp/model/attributes",
    "./pkg/otlp/model/attributes/azure",
    "./pkg/otlp/model/attributes/ec2",
    "./pkg/otlp/model/attributes/gcp",
    "./pkg/otlp/model/internal",
    "./pkg/otlp/model/internal/instrumentationlibrary",
    "./pkg/otlp/model/internal/testutils",
    "./pkg/otlp/model/internal/utils",
    "./pkg/otlp/model/translator",
    "./pkg/quantile",
    "./pkg/quantile/summary",
    "./pkg/util/winutil",
    "./pkg/util/winutil/iphelper",
    "./pkg/util/winutil/pdhutil",
}


def is_go_file(path):
    """Checks if file is a go file from the Agent code."""
    return (path.startswith("pkg") or path.startswith("cmd")) and path.endswith(".go")


# Exclude non go files
# Get the package for each file
targets = {"./" + os.path.dirname(path) for path in sys.argv[1:] if is_go_file(path)}

# Exclude list above
targets = targets - EXCLUDED_FOLDERS

if len(targets) == 0:
    sys.exit()

# Call invoke command
# We do this workaround since we can't do relative imports
cmd = f"inv -e vet --targets='{','.join(targets)}'"

try:
    subprocess.run(cmd, shell=True, check=True)
except subprocess.CalledProcessError:
    # Signal failure to pre-commit
    sys.exit(-1)
