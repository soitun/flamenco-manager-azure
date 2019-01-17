# Azure+Go test

This document may miss some steps. Just be warned.

The [Azure Batch API Basics](https://docs.microsoft.com/en-us/azure/batch/batch-api-basics)
document is a nice place to start reading about Azure Batch. This document is also called
"Develop large-scale parallel compute solutions with Batch" and "Developer features".


## Initial setup for Azure

- Install the [Azure CLI client](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-apt?view=azure-cli-latest).
- Run `az login` and log in via your browser. Later you can run `az account list` to show the same info.
  The `id` field in the shown JSON is the subscription ID used below.
- Create an Azure Resource Group. This will hold all resources, like virtual
  machines, shared storage, network configuration, etc. Also configure the
  Azure CLI to set the location and resource group as default so that you
  don't have to specify them all the time.

      az group create --location westeurope --name myResourceGroup

- Configure the above settings as defaults for future Azure CLI calls:

      az account set -s 'subscription ID'
      az configure --defaults location=westeurope group=myResourceGroup

- Create a Batch account with `az batch account create --name baflamenco`
  (but probably that name is already taken, so pick one that's unique for you).
  You don't need to have `ba` in the name, but we do in this document to
  make it clear what is referencing to the batch account.
- Create a Storage account with `az storage account create --name saflamenco`
  You don't need to have `sa` in the name, but we do in this document to
  make it clear what is referencing to the storage account.
- Update `azure_config.json` with the information you used in the previous
  steps. See the `AZConfig` struct in `azconfig/azconfig.json` for explanation.


## Azure VM side of things

Create a VM for Flamenco Manager:

    az vm create --name vm-flamenco-manager --image Canonical:UbuntuServer:18.04-LTS:18.04.201901140 --admin-username azureuser --ssh-key-value "$(< ~/.ssh/id_rsa.pub)"

To get the IP address later, use:

    az network public-ip list --query [].ipAddress

To replace the SSH key that's allowed to log in on the VM, use:

    az vm user update -u azureuser --ssh-key-value "$(< ~/.ssh/id_rsa.pub)" -n vm-flamenco-manager

where `vm-flamenco-manager` is the name of the VM created earlier.

List your virtual network subnet IDs like this:

    az network vnet list

Update `azure_batch_pool.json` so that it has the default subnet ID configured in the `networkConfiguration/subnetId` key.


## Installing Flamenco Manager on the VM

On the above-installed VM, run the following commands (source: [MongoDB installation manual](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-ubuntu/))
as root:

    apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 9DA31620334BD75D9DCB49F368818C72E52529D4
    echo "deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu bionic/mongodb-org/4.0 multiverse" > /etc/apt/sources.list.d/mongodb-org-4.0.list
    apt-get update
    apt-get install -y mongodb-org
    systemctl enable mongod.service
    systemctl start mongod.service


## Azure Batch Explorer side of things

- Create application packages for Blender, FFMpeg, and Flamenco Worker.
    - MUST be ZIP files. Remember that ZIP files do not support symlinks.
    - The ZIP file MUST NOT contain a subdirectory for all the files; these are
      already created by Azure Batch. In other words, the `blender`, `ffmpeg`,
      and `flamenco-worker` executables should be at the top of the ZIP file.
    - Use `blender`, `ffmpeg`, and `flamenco-worker` as package IDs.
    - Edit the packages to have the version you just created as default version.

  My guess is that part of the VM's "startup task" is to get and extract those
  zip files into `/mnt/batch/tasks/applications` before running the pool-
  provided startup task. This is based on a trivial startup task taking around
  40 seconds to run.


## Get going with this Go code

- Run `az ad sp create-for-rbac --sdk-auth > client_credentials.json`
- Run `export AZURE_AUTH_LOCATION=$(pwd)/client_credentials.json`

- Install [dep](https://github.com/golang/dep#installation)
- Run `dep ensure`

To run this example, run:

    rsync -va flamenco-worker* flamenco.io:flamenco.io/azure/ && go install && azure-go-test -debug


## To get more info

- In `azbatch/azbatch.go` ctrl-click (or otherwise go do the definition of) the
  return type `batch.PoolAddParameter` of the `poolParameters()` function. This
  shows you what you can put into `azure_batch_pool.json`.


## On the VM

- The files are in `/mnt/batch/tasks`:
    - `/mnt/batch/tasks/applications`: zipped and unzipped application packages.
      Note that these are suffixed with a datetime (I'm guessing node startup),
      so use the environment variables (see below) to refer to them.
    - `/mnt/batch/tasks/startup/std{out,err}.txt`: stdout and stderr output of
      the startup task.
    - `/mnt/batch/tasks/startup/wd`: default work directory of the startup task.
    - `/mnt/batch/tasks/shared`: this is *not* shared between VMs, but shared
      between tasks on that VM.
    - For more info see [Files and Directories](https://docs.microsoft.com/en-us/azure/batch/batch-api-basics#files-and-directories).
- `/var/lib/waagent` contains info from the Azure Agent, like the assigned
  hostname, configuration settings, TLS certificates, etc.
- If the pool is configured to run the startup task as `NonAdmin`, it uses
  uid=1001(_azbatchtask_start) gid=1000(_azbatchgrp) groups=1000(_azbatchgrp).

These environment variables are set during the startup task:

    AZ_BATCH_ACCOUNT_NAME=flamenco
    AZ_BATCH_ACCOUNT_URL=https://flamenco.westeurope.batch.azure.com/
    AZ_BATCH_APP_PACKAGE_blender=/mnt/batch/tasks/applications/blender2.80-daily-2019-10-142019-01-15-09-19
    AZ_BATCH_APP_PACKAGE_blender_2_80_daily_2019_10_14=/mnt/batch/tasks/applications/blender2.80-daily-2019-10-142019-01-15-09-19
    AZ_BATCH_APP_PACKAGE_ffmpeg=/mnt/batch/tasks/applications/ffmpeg4.12019-01-15-09-16
    AZ_BATCH_APP_PACKAGE_ffmpeg_4_1=/mnt/batch/tasks/applications/ffmpeg4.12019-01-15-09-16
    AZ_BATCH_APP_PACKAGE_flamenco_worker=/mnt/batch/tasks/applications/flamenco-worker2.2.12019-01-15-08-52
    AZ_BATCH_APP_PACKAGE_flamenco_worker_2_2_1=/mnt/batch/tasks/applications/flamenco-worker2.2.12019-01-15-08-52
    AZ_BATCH_CERTIFICATES_DIR=/mnt/batch/tasks/startup/certs
    AZ_BATCH_NODE_ID=tvm-383584635_1-20190115t092314z
    AZ_BATCH_NODE_IS_DEDICATED=true
    AZ_BATCH_NODE_ROOT_DIR=/mnt/batch/tasks
    AZ_BATCH_NODE_SHARED_DIR=/mnt/batch/tasks/shared
    AZ_BATCH_NODE_STARTUP_DIR=/mnt/batch/tasks/startup
    AZ_BATCH_POOL_ID=je-moeder-47
    AZ_BATCH_TASK_DIR=/mnt/batch/tasks/startup
    AZ_BATCH_TASK_USER=_azbatchtask_start
    AZ_BATCH_TASK_USER_IDENTITY=TaskNonAdmin
    AZ_BATCH_TASK_WORKING_DIR=/mnt/batch/tasks/startup/wd
    HOME=/mnt/batch/tasks/startup/wd
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/mnt/batch/tasks/shared:/mnt/batch/tasks/startup/wd
    PWD=/mnt/batch/tasks/startup/wd
    SHLVL=1
    USER=_azbatchtask_start
    _=/usr/bin/env
