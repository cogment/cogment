# aom-orchestrator

# Hacking on the orchestrator

```
# Launch a dev_env docker container with the framework source mounted
me@machine:~/git/aom-framework/orchestrator$ ./dev_env.sh

# Build the orchestrator
bash-4.4# mkdir _bld
bash-4.4# cd _bld
bash-4.4# cmake ..
bash-4.4# make -j $(nproc) aom_orchestrator

# Run tests
bash-4.4# ctest

# Launch the orchestrator
bash-4.4# ./orchestrator/aom_orchestrator
```