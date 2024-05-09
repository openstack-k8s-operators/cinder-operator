# Configure Cinder with Ceph backend

The Cinder services can be configured to interact with an external Ceph cluster.
In particular, the `customServiceConfig` parameter must be used, for each defined
`cinder-volume` and `cinder-backup` instance, to override the `enabled_backends`
parameter and inject the Ceph related parameters.
The `ceph.conf` and the `client keyring` must exist as secrets, and can be
mounted by the cinder pods using the `extraMounts` feature.

Create a secret by generating the following file and then apply it using the `oc`
cli.

---
apiVersion: v1
kind: Secret
metadata:
  name: ceph-client-conf
  namespace: openstack
stringData:
  ceph.client.openstack.keyring: |
    [client.openstack]
        key = <secret key>
        caps mgr = "allow *"
        caps mon = "profile rbd"
        caps osd = "profile rbd pool=images"
  ceph.conf: |
    [global]
    fsid = 7a1719e8-9c59-49e2-ae2b-d7eb08c695d4
    mon_host = 10.1.1.2,10.1.1.3,10.1.1.4


Add the following to the spec of the Cinder CR and then apply it using the `oc`
cli.

```
  extraMounts:
    - name: v1
      region: r1
      extraVol:
        - propagation:
          - CinderVolume
          - CinderBackup
          volumes:
          - name: ceph
            secret:
              secretName: ceph-client-conf
          mounts:
          - name: ceph
            mountPath: "/etc/ceph"
            readOnly: true
```

The following represents an example of the entire Cinder object that can be used
to trigger the Cinder service deployment, and enable the Cinder backend that
points to an external Ceph cluster.


```
apiVersion: cinder.openstack.org/v1beta1
kind: Cinder
metadata:
  name: cinder
  namespace: openstack
spec:
  serviceUser: cinder
  databaseInstance: openstack
  databaseUser: cinder
  cinderAPI:
    replicas: 1
    containerImage: quay.io/podified-antelopecentos9/openstack-cinder-api:current-podified
  cinderScheduler:
    replicas: 1
    containerImage: quay.io/podified-antelopecentos9/openstack-cinder-scheduler:current-podified
  cinderBackup:
    replicas: 1
    containerImage: quay.io/podified-antelopecentos9/openstack-cinder-backup:current-podified
    customServiceConfig: |
      [DEFAULT]
      backup_driver = cinder.backup.drivers.ceph.CephBackupDriver
      backup_ceph_pool = backups
      backup_ceph_user = admin
  secret: cinder-secret
  cinderVolumes:
    volume1:
      containerImage: quay.io/podified-antelopecentos9/openstack-cinder-volume:current-podified
      replicas: 1
      customServiceConfig: |
        [DEFAULT]
        enabled_backends=ceph
        [ceph]
        volume_backend_name=ceph
        volume_driver=cinder.volume.drivers.rbd.RBDDriver
        rbd_ceph_conf=/etc/ceph/ceph.conf
        rbd_user=admin
        rbd_pool=volumes
        rbd_flatten_volume_from_snapshot=False
        rbd_secret_uuid=<Ceph_FSID>
  extraMounts:
    - name: cephfiles
      region: r1
      extraVol:
      - propagation:
        - CinderVolume
        - CinderBackup
        extraVolType: Ceph
        volumes:
        - name: ceph
          secret:
            secretName: ceph-client-conf
        mounts:
        - name: ceph
          mountPath: "/etc/ceph"
          readOnly: true
```

When the service is up and running, it's possible to interact with the Cinder
API and create the Ceph `cinder type` backend which is associated with the Ceph
tier specified in the config file.
