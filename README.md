# thermo-center-controller

Kubernetes controller for [Thermo-Center](https://github.com/rkojedzinszky/thermo-center).

## Deployment

Deploy the Custom Resource Definition:

```shell
$ kubectl apply -f https://raw.githubusercontent.com/rkojedzinszky/thermo-center-controller/master/config/crd/kojedz.in_thermocenters.yaml
```

Then, create a dedicated namespace for Thermo-Center:

```shell
$ kubectl create ns thermo-center
```

Deploy the operator:

```shell
$ kubectl -n thermo-center apply -f https://raw.githubusercontent.com/rkojedzinszky/thermo-center-controller/master/config/rbac/role.yaml \
  -f https://raw.githubusercontent.com/rkojedzinszky/thermo-center-controller/master/deploy/service_account.yaml \
  -f https://raw.githubusercontent.com/rkojedzinszky/thermo-center-controller/master/deploy/role_binding.yaml \
  -f https://raw.githubusercontent.com/rkojedzinszky/thermo-center-controller/master/deploy/controller.yaml
```

Now the operator is up and running.

Follow setup instructions [here](https://github.com/rkojedzinszky/thermo-center/tree/master/deploy/kubernetes#spi-devicenode-setup) to have a working radio module. Also prepare an empty PostgreSQL database. Then, deploy thermo-center customizing the following CRD:

```yaml
apiVersion: kojedz.in/v1alpha1
kind: ThermoCenter
metadata:
  name: thermo-center
spec:
  database:
    host: postgres.db
    name: thermo-center
    password: thermo-center-password
    port: 5432
    user: thermo-center
  ingress:
    hostNames:
    - your.domain.name
  replicas: 1
  version: 3.3.1
```

Apply it, and then, create a superuser as:
```shell
$ ns=thermo-center; kubectl -n $ns exec -it $(kubectl -n $ns get pod -l thermo-center-component=api --template '{{(index .items 0).metadata.name}}') -- python manage.py createsuperuser
```

Then you will be able to access your installation at http://your.domain.name .
