### Definitions

- Public Workstation: A laptop somewhere, has access to internet, probably in a DMZ somewhere. In this case we'll use a GCP server, `dex-airgap-jump`
- Airgapped Workstation: `dex-airgap-workstation` -- has kubectl access to an "airgapped cluster", and network access to a "private registry"
- Airgapped Cluster: in this case we'll [use a kURL cluster in GCP](#appendix-creating-an-airgapped-kubernetes-cluster-in-gcp), but any k8s cluster works, if you want a real test you should remove any public outbound internet gateways though. This example uses a node called `dex-airgap-cluster`
- Private Registry: a separate registry to which images will be pushed during install, and pulled from within the cluster. The cluster should have network access to this registry to pull images.
- KOTS Bundle: Kots bundle can be [built from source](#appendix-building-the-bundle), or downloaded from s3: https://kots-experimental.s3.amazonaws.com/kots-v1.16.1-airgap-experimental-alpha2.tar.gz

### Installing


From Public Workstation, move kots bundle to Airgapped Workstation

```
gcloud compute scp kots-v1.16.1.tar.gz dex-airgap-jump:
```

This installer expects a namespace and a pull secret to already exist on the target cluster. Let's create them from the Airgapped Workstation

```
export NAMESPACE=test-deploy
kubectl create namespace ${NAMESPACE}
```

```
namespace/test-deploy created
```

we also need to create an image pull secret:

```
export DOCKER_REGISTRY=...
export DOCKER_USERNAME=...
export DOCKER_PASSWORD=...
kubectl -n $NAMESPACE create secret  docker-registry registry-creds --docker-server=$DOCKER_REGISTRY --docker-username=$DOCKER_USERNAME --docker-password=$DOCKER_PASSWORD --docker-email=a@b.c
```

```
secret/registry-creds created
```

Now, let's unpack the bundle and run the script with no arguments to see the usage:

```
tar xvf kots-*.tar.gz
./install.sh
```
You should see an error and some help text

```
error: missing registry URL


USAGE
install.sh: load a kotsadm bundle


Positional Arguments

KOTS_REGISTRY_URL -- the url to a registry to use, with any path attached
KOTS_NAMESPACE -- the name of an existing namespace to deploy into
KOTS_IMAGE_PULL_SECRET_NAME -- the name of an existing image pull secret to use for pulling from the private registry

Optional Arguments:

KOTS_REGISTRY_USERNAME -- optional, username to push images. Leave blank if this workstation already had docker push access
KOTS_REGISTRY_PASSWORD -- optional, password to push images. Leave blank if this workstation already had docker push access

```

Go to the directory yaml and open the file kotsadm.yaml for editing.

Search for KOTSADM_API_ENDPOINT and you should see something like: 
```
- name: KOTSADM_API_ENDPOINT
    value: http://kotsadm-api-node.appname.svc.cluster.local:3000
```
You need to update the value to replace appname with the namespace you have created to deploy harness.
```
value: http://kotsadm-api-node.<namespace>.svc.cluster.local:3000
```

Let's invoke this with our set variables (I'm leaving namespace blank in this case). Since my machine is already configured to push to the registry via `docker login`,
I'll omit the docker credentials.

```shell script
./install.sh "${DOCKER_REGISTRY}" "${NAMESPACE}" "registry-creds" 
```

If you need to pass credentials you can add them with two additional arguments

```shell script
./install.sh "${DOCKER_REGISTRY}" "${NAMESPACE}" "registry-creds" "${DOCKER_USERNAME}" "${DOCKER_PASSWORD}"
```


Once this is finished and the postflight checks have completed, we can get a new password for the admin console:

```shell script
./kots reset-password -n "${NAMESPACE}"
```

Next, we need to expose the admin console. If we're running kubectl from a workstation with a browser, we can run 

```shell script
./kots admin-console -n "${NAMESPACE}"
```

Since I'm running this from a Jump box, I'm going to create a NodePort, but other options like creating an Ingress will let us configure TLS, etc. For an Ingress the service name should be `kotsadm` and it can be deployed out-of-band, either before or after the full install. If you're

```text
$ kubectl -n "${NAMESPACE}" expose deployment kotsadm --name=kotsadm-nodeport --port=3000 --target-port=3000 --type=NodePort
$ kubectl -n "${NAMESPACE}" get svc kotsadm-nodeport
NAME               TYPE       CLUSTER-IP    EXTERNAL-IP   PORT(S)          AGE
kotsadm-nodeport   NodePort   10.96.0.239   <none>        3000:25124/TCP   7s
```

And then navigate to <instance ip> : <port>, in this case `http://25.238.234.48:25124`

**Note**: in this case we've cheated and given the "airgapped cluster" a public IP so we can reach the kotsadm console, but you can check out the [end to end gcp example](./end_to_end_gcp_example.md) for an example of using an airgapped registry and an ssh tunnel for a "full airgap" example where instances don't have internet gateways. If your airgapped cluster has access via a VPN, that's an option as well.
  


