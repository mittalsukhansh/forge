import os
import subprocess
import yaml

with open("buildspec.yml") as file:
    spec = yaml.safe_load(file)

env = os.environ.copy()
env.update(spec["env"]["variables"])
env.pop("CODEBUILD_RESOLVED_SOURCE_VERSION", None)

commands = [
    "set -euo pipefail",
    "rm -f /tmp/forge-build-success"
]

for phase in ["install", "pre_build", "build", "post_build"]:
    commands.append(
        f'echo "========== {phase.upper()} =========="'
    )
    commands.extend(spec["phases"][phase]["commands"])

subprocess.run(
    ["bash", "-c", "\n".join(commands)],
    env=env,
    check=True
)
