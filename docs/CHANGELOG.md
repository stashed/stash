---
title: Changelog | Stash
description: Changelog
menu:
  product_stash_0.8.2:
    identifier: changelog-stash
    name: Changelog
    parent: welcome
    weight: 10
product_name: stash
menu_name: product_stash_0.8.2
section_menu_id: welcome
url: /products/stash/0.8.2/welcome/changelog/
aliases:
  - /products/stash/0.8.2/CHANGELOG/
---

# Change Log

## [0.8.2](https://github.com/appscode/stash/tree/0.8.2) (2019-01-02)
[Full Changelog](https://github.com/appscode/stash/compare/0.8.1...0.8.2)

**Fixed bugs:**

- Fix typo in installer [\#638](https://github.com/appscode/stash/pull/638) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Backend configuration doc link broken [\#640](https://github.com/appscode/stash/issues/640)
- Architecture questions [\#635](https://github.com/appscode/stash/issues/635)
- Restart operator pod on update [\#611](https://github.com/appscode/stash/issues/611)

**Merged pull requests:**

- Prepare docs for 0.8.2 release [\#644](https://github.com/appscode/stash/pull/644) ([tamalsaha](https://github.com/tamalsaha))
- Update copyright notice for 2019 [\#643](https://github.com/appscode/stash/pull/643) ([tamalsaha](https://github.com/tamalsaha))
- Use stash.labels template in chart [\#642](https://github.com/appscode/stash/pull/642) ([tamalsaha](https://github.com/tamalsaha))
- Fixed broken link for bakend [\#641](https://github.com/appscode/stash/pull/641) ([hossainemruz](https://github.com/hossainemruz))
-  Only mount stash apiserver `tls.crt` into Prometheus [\#639](https://github.com/appscode/stash/pull/639) ([hossainemruz](https://github.com/hossainemruz))
- Fix monitoring in helm + update doc to match with third-party-tools tutorial [\#637](https://github.com/appscode/stash/pull/637) ([hossainemruz](https://github.com/hossainemruz))
- Add certificate health checker [\#636](https://github.com/appscode/stash/pull/636) ([tamalsaha](https://github.com/tamalsaha))
- Update chart readme [\#632](https://github.com/appscode/stash/pull/632) ([tamalsaha](https://github.com/tamalsaha))
- Update webhook error message format for Kubernetes 1.13+ [\#631](https://github.com/appscode/stash/pull/631) ([tamalsaha](https://github.com/tamalsaha))
- Fix typos [\#630](https://github.com/appscode/stash/pull/630) ([tamalsaha](https://github.com/tamalsaha))

## [0.8.1](https://github.com/appscode/stash/tree/0.8.1) (2018-12-09)
[Full Changelog](https://github.com/appscode/stash/compare/0.8.0...0.8.1)

**Fixed bugs:**

- Stash chart is throwing error [\#627](https://github.com/appscode/stash/issues/627)

**Merged pull requests:**

- Prepare docs for 0.8.1 release [\#629](https://github.com/appscode/stash/pull/629) ([tamalsaha](https://github.com/tamalsaha))
- Add missing validator for respository resource in chart [\#628](https://github.com/appscode/stash/pull/628) ([tamalsaha](https://github.com/tamalsaha))

## [0.8.0](https://github.com/appscode/stash/tree/0.8.0) (2018-12-08)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0...0.8.0)

**Fixed bugs:**

- Delete snapshot command does not check for snapshot's existence [\#549](https://github.com/appscode/stash/issues/549)
- Backup not triggered  [\#461](https://github.com/appscode/stash/issues/461)
- Service name hardcoded in func PushgatewayURL, no metrics available [\#596](https://github.com/appscode/stash/issues/596)
- Fix extended apiserver issues with Kubernetes 1.11 [\#536](https://github.com/appscode/stash/pull/536) ([tamalsaha](https://github.com/tamalsaha))
- Correctly handle ignored openapi prefixes [\#533](https://github.com/appscode/stash/pull/533) ([tamalsaha](https://github.com/tamalsaha))
- Add rbac permissions for snapshots [\#531](https://github.com/appscode/stash/pull/531) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Problem creating backups [\#588](https://github.com/appscode/stash/issues/588)
- Issue while installing stash kubernetes 1.11.2 [\#587](https://github.com/appscode/stash/issues/587)
- Hardcoded cleaner kubectl image in Helm chart [\#583](https://github.com/appscode/stash/issues/583)
- Deployed latest helm chart and getting error during sidecar creation [\#556](https://github.com/appscode/stash/issues/556)
- Minio backup fails: 'net/http: invalid header field value "..." for key Authorization' [\#547](https://github.com/appscode/stash/issues/547)
- Repository overwrite for different workload with same name in different namespace [\#539](https://github.com/appscode/stash/issues/539)
- Unexpected behavior in offline backup [\#535](https://github.com/appscode/stash/issues/535)
- Offline backup not working \(permissions\) [\#534](https://github.com/appscode/stash/issues/534)
- Support node selector for recovery job [\#515](https://github.com/appscode/stash/issues/515)
- Clarify that hostpaths are just example [\#514](https://github.com/appscode/stash/issues/514)
- Internal error occurred: failed calling admission webhook "deployment.admission.stash.appscode.com": the server could not find the requested resource [\#510](https://github.com/appscode/stash/issues/510)
- GKE page missing front matter [\#505](https://github.com/appscode/stash/issues/505)
- Could not list snapshots on kubernetes 1.8.4 [\#503](https://github.com/appscode/stash/issues/503)
- Admission webhook denied rquest: Rolebindings not found [\#501](https://github.com/appscode/stash/issues/501)
- Incorrect image name for sidecar container [\#485](https://github.com/appscode/stash/issues/485)
- Using Stash with TLS secured Minio Server Can't succeed [\#478](https://github.com/appscode/stash/issues/478)
- Add cluster name in repo path [\#374](https://github.com/appscode/stash/issues/374)
- Stash don't pass `nodeSelector` from Recovery crd to recovery Job. [\#617](https://github.com/appscode/stash/issues/617)
- Permissions problem with the Helm chart in master branch [\#592](https://github.com/appscode/stash/issues/592)
- Add Prometheus config sample for pushgateway [\#582](https://github.com/appscode/stash/issues/582)
- Handle security context [\#566](https://github.com/appscode/stash/issues/566)
- \[Request\] Add backup details to "kubectl get" for stash objects on K8s 1.11 [\#525](https://github.com/appscode/stash/issues/525)
- matchLabels on Restic CRD not working when using hyphens in keys [\#521](https://github.com/appscode/stash/issues/521)

**Merged pull requests:**

- Prepare docs for 0.8.0 release [\#626](https://github.com/appscode/stash/pull/626) ([tamalsaha](https://github.com/tamalsaha))
- Update docs \(Minio, Rook, NFS\) [\#625](https://github.com/appscode/stash/pull/625) ([hossainemruz](https://github.com/hossainemruz))
- Use flags.DumpAll to dump flags [\#624](https://github.com/appscode/stash/pull/624) ([tamalsaha](https://github.com/tamalsaha))
- Set periodic analytics [\#623](https://github.com/appscode/stash/pull/623) ([tamalsaha](https://github.com/tamalsaha))
- Fix e2e test [\#622](https://github.com/appscode/stash/pull/622) ([hossainemruz](https://github.com/hossainemruz))
- Recovery Job: Use nodeName for DaemonSet and nodeSelector for other workloads [\#620](https://github.com/appscode/stash/pull/620) ([hossainemruz](https://github.com/hossainemruz))
- Pass --enable-\*\*\*-webhook flags to operator [\#619](https://github.com/appscode/stash/pull/619) ([tamalsaha](https://github.com/tamalsaha))
- Add validation webhook xray [\#618](https://github.com/appscode/stash/pull/618) ([tamalsaha](https://github.com/tamalsaha))
- Use dynamic pushgateway url [\#614](https://github.com/appscode/stash/pull/614) ([hossainemruz](https://github.com/hossainemruz))
- Add docs for AKS and EKS [\#609](https://github.com/appscode/stash/pull/609) ([hossainemruz](https://github.com/hossainemruz))
- Improve monitoring facility [\#606](https://github.com/appscode/stash/pull/606) ([hossainemruz](https://github.com/hossainemruz))
- Pass image pull secrets for cleaner job in chart [\#598](https://github.com/appscode/stash/pull/598) ([tamalsaha](https://github.com/tamalsaha))
- Update kubernetes client libraries to 1.12.0 [\#597](https://github.com/appscode/stash/pull/597) ([tamalsaha](https://github.com/tamalsaha))
- Support LogLevel in chart [\#594](https://github.com/appscode/stash/pull/594) ([tamalsaha](https://github.com/tamalsaha))
- Check if Kubernetes version is supported before running operator [\#593](https://github.com/appscode/stash/pull/593) ([tamalsaha](https://github.com/tamalsaha))
- Enable webhooks by default in chart [\#591](https://github.com/appscode/stash/pull/591) ([tamalsaha](https://github.com/tamalsaha))
- Update chart readme for cleaner values [\#590](https://github.com/appscode/stash/pull/590) ([tamalsaha](https://github.com/tamalsaha))
- Fix \#583 and pushgateway version [\#584](https://github.com/appscode/stash/pull/584) ([sebastien-prudhomme](https://github.com/sebastien-prudhomme))
- Use --pull flag with docker build [\#581](https://github.com/appscode/stash/pull/581) ([tamalsaha](https://github.com/tamalsaha))
- Use kubernetes-1.11.3 [\#578](https://github.com/appscode/stash/pull/578) ([tamalsaha](https://github.com/tamalsaha))
- Update CertStore [\#576](https://github.com/appscode/stash/pull/576) ([tamalsaha](https://github.com/tamalsaha))
- Use apps/v1 apigroup in installer scripts [\#574](https://github.com/appscode/stash/pull/574) ([tamalsaha](https://github.com/tamalsaha))
- Support pod annotations in chart [\#573](https://github.com/appscode/stash/pull/573) ([tamalsaha](https://github.com/tamalsaha))
- Set serviceAccount for clearner job [\#572](https://github.com/appscode/stash/pull/572) ([tamalsaha](https://github.com/tamalsaha))
- Set SecurityContext for stash sidecar [\#570](https://github.com/appscode/stash/pull/570) ([tamalsaha](https://github.com/tamalsaha))
- Cleanup webhooks when chart is deleted [\#569](https://github.com/appscode/stash/pull/569) ([tamalsaha](https://github.com/tamalsaha))
- Use IntHash as status.observedGeneration [\#568](https://github.com/appscode/stash/pull/568) ([tamalsaha](https://github.com/tamalsaha))
- fix success list in grafana dashboard [\#567](https://github.com/appscode/stash/pull/567) ([unteem](https://github.com/unteem))
- Update pipeline [\#565](https://github.com/appscode/stash/pull/565) ([tahsinrahman](https://github.com/tahsinrahman))
- Add observedGenerationHash field [\#564](https://github.com/appscode/stash/pull/564) ([tamalsaha](https://github.com/tamalsaha))
- fix uninstall for concourse [\#563](https://github.com/appscode/stash/pull/563) ([tahsinrahman](https://github.com/tahsinrahman))
- Fix chart values file [\#562](https://github.com/appscode/stash/pull/562) ([tamalsaha](https://github.com/tamalsaha))
- Improve Helm chart options [\#561](https://github.com/appscode/stash/pull/561) ([tamalsaha](https://github.com/tamalsaha))
- Refactor concourse scripts [\#554](https://github.com/appscode/stash/pull/554) ([tahsinrahman](https://github.com/tahsinrahman))
- Add AlreadyObserved methods [\#553](https://github.com/appscode/stash/pull/553) ([tamalsaha](https://github.com/tamalsaha))
- Add categories support to crds [\#552](https://github.com/appscode/stash/pull/552) ([tamalsaha](https://github.com/tamalsaha))
- Improve logging [\#551](https://github.com/appscode/stash/pull/551) ([hossainemruz](https://github.com/hossainemruz))
- Improve doc [\#550](https://github.com/appscode/stash/pull/550) ([hossainemruz](https://github.com/hossainemruz))
- Check for snapshot existence before delete [\#548](https://github.com/appscode/stash/pull/548) ([hossainemruz](https://github.com/hossainemruz))
- Enable status sub resource for crd yamls [\#546](https://github.com/appscode/stash/pull/546) ([tamalsaha](https://github.com/tamalsaha))
- Retry UpdateStatus calls [\#544](https://github.com/appscode/stash/pull/544) ([tamalsaha](https://github.com/tamalsaha))
- Move crds to api folder [\#543](https://github.com/appscode/stash/pull/543) ([tamalsaha](https://github.com/tamalsaha))
- Revendor objectstore api [\#542](https://github.com/appscode/stash/pull/542) ([tamalsaha](https://github.com/tamalsaha))
- Use kmodules.xyz/objectstore-api [\#541](https://github.com/appscode/stash/pull/541) ([tamalsaha](https://github.com/tamalsaha))
- Fix offline backup [\#537](https://github.com/appscode/stash/pull/537) ([hossainemruz](https://github.com/hossainemruz))
- Rename dev script [\#532](https://github.com/appscode/stash/pull/532) ([tamalsaha](https://github.com/tamalsaha))
- Use version and additional columns for crds [\#530](https://github.com/appscode/stash/pull/530) ([tamalsaha](https://github.com/tamalsaha))
- Don't add admission/v1beta1 group as a prioritized version [\#529](https://github.com/appscode/stash/pull/529) ([tamalsaha](https://github.com/tamalsaha))
- Update client-go to v8.0.0 [\#528](https://github.com/appscode/stash/pull/528) ([tamalsaha](https://github.com/tamalsaha))
- Format shell scripts [\#526](https://github.com/appscode/stash/pull/526) ([tamalsaha](https://github.com/tamalsaha))
- Enable status subresource for crds [\#524](https://github.com/appscode/stash/pull/524) ([tamalsaha](https://github.com/tamalsaha))
- Upgrade to restic 0.9.1 [\#522](https://github.com/appscode/stash/pull/522) ([tamalsaha](https://github.com/tamalsaha))
- Move openapi-spec to api folder [\#513](https://github.com/appscode/stash/pull/513) ([tamalsaha](https://github.com/tamalsaha))
- Deploy operator in kube-system namespace via Helm [\#511](https://github.com/appscode/stash/pull/511) ([tamalsaha](https://github.com/tamalsaha))
- Add togglable tabs for Installation: Script & Helm [\#509](https://github.com/appscode/stash/pull/509) ([sajibcse68](https://github.com/sajibcse68))
- Revendor dependencies [\#508](https://github.com/appscode/stash/pull/508) ([tamalsaha](https://github.com/tamalsaha))
- Added front matter [\#507](https://github.com/appscode/stash/pull/507) ([hossainemruz](https://github.com/hossainemruz))
- Improve installer [\#504](https://github.com/appscode/stash/pull/504) ([tamalsaha](https://github.com/tamalsaha))
- Use apps/v1 apigroup [\#555](https://github.com/appscode/stash/pull/555) ([tamalsaha](https://github.com/tamalsaha))
- Update chart installation instruction for Kubernetes 1.11 [\#527](https://github.com/appscode/stash/pull/527) ([tamalsaha](https://github.com/tamalsaha))
- Remove status from crd.yaml [\#523](https://github.com/appscode/stash/pull/523) ([tamalsaha](https://github.com/tamalsaha))
- Upgrade to prom/pushgateway:v0.5.2 [\#519](https://github.com/appscode/stash/pull/519) ([tamalsaha](https://github.com/tamalsaha))
- Remove ops-address port [\#518](https://github.com/appscode/stash/pull/518) ([tamalsaha](https://github.com/tamalsaha))
- Set cpu limits to 100m [\#517](https://github.com/appscode/stash/pull/517) ([tamalsaha](https://github.com/tamalsaha))
- Support node selector for recovery job [\#516](https://github.com/appscode/stash/pull/516) ([tamalsaha](https://github.com/tamalsaha))
- Fix concourse test [\#496](https://github.com/appscode/stash/pull/496) ([hossainemruz](https://github.com/hossainemruz))

## [0.7.0](https://github.com/appscode/stash/tree/0.7.0) (2018-05-29)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.5...0.7.0)

**Implemented enhancements:**

- Support custom CA cert with backend [\#288](https://github.com/appscode/stash/issues/288)

**Fixed bugs:**

- Pod restart after each backup when Mutating Webhook enabled [\#396](https://github.com/appscode/stash/issues/396)
- Sidecar RoleBinding is not being created when Mutating Webhook is enabled  [\#395](https://github.com/appscode/stash/issues/395)
- Recovery to PVC restores data in subdirectory instead of root directory [\#392](https://github.com/appscode/stash/issues/392)
- Forget panics in 0.7.0-rc.0 [\#373](https://github.com/appscode/stash/issues/373)

**Closed issues:**

- Resource type "snapshot" not registered [\#499](https://github.com/appscode/stash/issues/499)
- Support Repository deletion [\#416](https://github.com/appscode/stash/issues/416)
- Docs TODO [\#414](https://github.com/appscode/stash/issues/414)
- Convert Initializer to MutationWebhook [\#326](https://github.com/appscode/stash/issues/326)
- Use informer factory for backup scheduler [\#321](https://github.com/appscode/stash/issues/321)
- Show repository snapshot list [\#319](https://github.com/appscode/stash/issues/319)
- Verbosity \(--v\) flag not inherited to backup sidecars [\#282](https://github.com/appscode/stash/issues/282)
- Double Deployment patch when deleting a Restic CRD? [\#281](https://github.com/appscode/stash/issues/281)
- Consider a simple 'enabled' switch for Restic CRD [\#279](https://github.com/appscode/stash/issues/279)
- offline backup is not supported for workload kind `Deployment`, `Replicaset` and `ReplicationController` with `replicas \> 1` [\#244](https://github.com/appscode/stash/issues/244)
- Recover specific snapshot ID [\#215](https://github.com/appscode/stash/issues/215)

**Merged pull requests:**

- Prepare docs for 0.7.0 release. [\#502](https://github.com/appscode/stash/pull/502) ([tamalsaha](https://github.com/tamalsaha))
- Set RollingUpdate for DaemonSet [\#349](https://github.com/appscode/stash/pull/349) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.5](https://github.com/appscode/stash/tree/0.7.0-rc.5) (2018-05-23)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.4...0.7.0-rc.5)

**Fixed bugs:**

- Fix storage implementation for snapshots [\#497](https://github.com/appscode/stash/pull/497) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Prepare docs for 0.7.0-rc.5 [\#498](https://github.com/appscode/stash/pull/498) ([tamalsaha](https://github.com/tamalsaha))
- Update changelog [\#495](https://github.com/appscode/stash/pull/495) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.4](https://github.com/appscode/stash/tree/0.7.0-rc.4) (2018-05-22)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.3...0.7.0-rc.4)

**Fixed bugs:**

- Restic sidecar not properly working because of image tag error [\#443](https://github.com/appscode/stash/issues/443)
- Removed owner reference from repo-reader role-binding [\#484](https://github.com/appscode/stash/pull/484) ([hossainemruz](https://github.com/hossainemruz))
- Permit stash operator to perform pods/exec [\#433](https://github.com/appscode/stash/pull/433) ([tamalsaha](https://github.com/tamalsaha))
- Add missing batch jobs get RBAC permission [\#419](https://github.com/appscode/stash/pull/419) ([galexrt](https://github.com/galexrt))

**Closed issues:**

- Stash restore pod fails with istio sidecar [\#475](https://github.com/appscode/stash/issues/475)
- Stash stores GCS credentials in /tmp with 644 permissions [\#470](https://github.com/appscode/stash/issues/470)
- Update minio doc for 1.10? [\#467](https://github.com/appscode/stash/issues/467)
- Fix docs for StatefulSet [\#444](https://github.com/appscode/stash/issues/444)

**Merged pull requests:**

- Delete user roles on purge. [\#494](https://github.com/appscode/stash/pull/494) ([tamalsaha](https://github.com/tamalsaha))
- Add app: stash label to user roles. [\#493](https://github.com/appscode/stash/pull/493) ([tamalsaha](https://github.com/tamalsaha))
- Use post-install hooks to install admission controller in chart [\#492](https://github.com/appscode/stash/pull/492) ([tamalsaha](https://github.com/tamalsaha))
- Update changelog [\#491](https://github.com/appscode/stash/pull/491) ([tamalsaha](https://github.com/tamalsaha))
- Avoid creating apiservice when webhooks are not used. [\#490](https://github.com/appscode/stash/pull/490) ([tamalsaha](https://github.com/tamalsaha))
- Install correct version of stash chart [\#489](https://github.com/appscode/stash/pull/489) ([tamalsaha](https://github.com/tamalsaha))
- Use wait-until instead of fixed delay  [\#488](https://github.com/appscode/stash/pull/488) ([hossainemruz](https://github.com/hossainemruz))
- Concourse [\#486](https://github.com/appscode/stash/pull/486) ([tahsinrahman](https://github.com/tahsinrahman))
- Prepare docs for 0.7.0-rc.4 [\#483](https://github.com/appscode/stash/pull/483) ([tamalsaha](https://github.com/tamalsaha))
- Revendor [\#481](https://github.com/appscode/stash/pull/481) ([tamalsaha](https://github.com/tamalsaha))
- Fix enableRBAC  flag for sidecar [\#480](https://github.com/appscode/stash/pull/480) ([hossainemruz](https://github.com/hossainemruz))
- Typo \(`Weclome` → `Welcome`\) in page title [\#479](https://github.com/appscode/stash/pull/479) ([eliasp](https://github.com/eliasp))
- Add support for initial backoff to the apiserver call on recover [\#476](https://github.com/appscode/stash/pull/476) ([farcaller](https://github.com/farcaller))
- Support recovering from repository in different namespace [\#474](https://github.com/appscode/stash/pull/474) ([tamalsaha](https://github.com/tamalsaha))
- Update docs \(run minio in v1.9.4+ cluster and add example yaml files in respective backends\) [\#473](https://github.com/appscode/stash/pull/473) ([hossainemruz](https://github.com/hossainemruz))
- Limit the GCS file permissions to owner only [\#472](https://github.com/appscode/stash/pull/472) ([farcaller](https://github.com/farcaller))
- Fix a typo [\#471](https://github.com/appscode/stash/pull/471) ([farcaller](https://github.com/farcaller))
- Don't panic if admission options is nil [\#469](https://github.com/appscode/stash/pull/469) ([tamalsaha](https://github.com/tamalsaha))
- Disable admission controllers for webhook server [\#468](https://github.com/appscode/stash/pull/468) ([tamalsaha](https://github.com/tamalsaha))
- Use new UpdateRecoveryStatus method [\#466](https://github.com/appscode/stash/pull/466) ([tamalsaha](https://github.com/tamalsaha))
- Add Update\*\*\*Status helpers [\#465](https://github.com/appscode/stash/pull/465) ([tamalsaha](https://github.com/tamalsaha))
- Added SSL support for deleting restic repository from Minio backend [\#464](https://github.com/appscode/stash/pull/464) ([hossainemruz](https://github.com/hossainemruz))
- Update client-go to 7.0.0 [\#463](https://github.com/appscode/stash/pull/463) ([tamalsaha](https://github.com/tamalsaha))
- Rename webhook files in chart [\#460](https://github.com/appscode/stash/pull/460) ([tamalsaha](https://github.com/tamalsaha))
- Update workload api [\#459](https://github.com/appscode/stash/pull/459) ([tamalsaha](https://github.com/tamalsaha))
- Remove stash crds before uninstalling operator [\#458](https://github.com/appscode/stash/pull/458) ([tamalsaha](https://github.com/tamalsaha))
- Export kube-ca only if required [\#457](https://github.com/appscode/stash/pull/457) ([tamalsaha](https://github.com/tamalsaha))
- Improve installer [\#456](https://github.com/appscode/stash/pull/456) ([tamalsaha](https://github.com/tamalsaha))
- Update changelog [\#455](https://github.com/appscode/stash/pull/455) ([tamalsaha](https://github.com/tamalsaha))
- Various installer fixes [\#454](https://github.com/appscode/stash/pull/454) ([tamalsaha](https://github.com/tamalsaha))
- Update workload client [\#453](https://github.com/appscode/stash/pull/453) ([tamalsaha](https://github.com/tamalsaha))
- Update workload client [\#452](https://github.com/appscode/stash/pull/452) ([tamalsaha](https://github.com/tamalsaha))
- Revendor workload client [\#451](https://github.com/appscode/stash/pull/451) ([tamalsaha](https://github.com/tamalsaha))
- Update workload api [\#450](https://github.com/appscode/stash/pull/450) ([tamalsaha](https://github.com/tamalsaha))
- Fixes RBAC permission for scaledownCronJob [\#449](https://github.com/appscode/stash/pull/449) ([hossainemruz](https://github.com/hossainemruz))
- Used Snapshot  to verify successful backup [\#447](https://github.com/appscode/stash/pull/447) ([hossainemruz](https://github.com/hossainemruz))
- Some cleanup [\#446](https://github.com/appscode/stash/pull/446) ([tamalsaha](https://github.com/tamalsaha))
- Update StatefulSet doc [\#445](https://github.com/appscode/stash/pull/445) ([hossainemruz](https://github.com/hossainemruz))
- pkg/util: fix error found by vet [\#442](https://github.com/appscode/stash/pull/442) ([functionary](https://github.com/functionary))
- Move Stash swagger.json to top level folder [\#441](https://github.com/appscode/stash/pull/441) ([tamalsaha](https://github.com/tamalsaha))
- Fix go\_vet error [\#440](https://github.com/appscode/stash/pull/440) ([hossainemruz](https://github.com/hossainemruz))
- Delete restic repository from backend if Repository CRD is deleted [\#438](https://github.com/appscode/stash/pull/438) ([hossainemruz](https://github.com/hossainemruz))
- Recover specific snapshot [\#437](https://github.com/appscode/stash/pull/437) ([hossainemruz](https://github.com/hossainemruz))
- Use Repository data in Recovery CRD [\#436](https://github.com/appscode/stash/pull/436) ([hossainemruz](https://github.com/hossainemruz))
- Increase qps and burst limits [\#435](https://github.com/appscode/stash/pull/435) ([tamalsaha](https://github.com/tamalsaha))
- Add RBAC instructions for GKE cluster [\#432](https://github.com/appscode/stash/pull/432) ([tamalsaha](https://github.com/tamalsaha))
- Update charts location [\#431](https://github.com/appscode/stash/pull/431) ([tamalsaha](https://github.com/tamalsaha))
- Add docs for GKE and Rook [\#430](https://github.com/appscode/stash/pull/430) ([hossainemruz](https://github.com/hossainemruz))
- concourse configs [\#429](https://github.com/appscode/stash/pull/429) ([tahsinrahman](https://github.com/tahsinrahman))
- Skip lock while listing snapshots [\#428](https://github.com/appscode/stash/pull/428) ([hossainemruz](https://github.com/hossainemruz))
- Purge repository objects in installer [\#427](https://github.com/appscode/stash/pull/427) ([tamalsaha](https://github.com/tamalsaha))
- Support installing from local installer scripts [\#426](https://github.com/appscode/stash/pull/426) ([tamalsaha](https://github.com/tamalsaha))
- Fixed Repository yaml in doc [\#425](https://github.com/appscode/stash/pull/425) ([hossainemruz](https://github.com/hossainemruz))
- Add delete method for snapshots to swagger.json [\#424](https://github.com/appscode/stash/pull/424) ([tamalsaha](https://github.com/tamalsaha))
- Generate swagger.json [\#423](https://github.com/appscode/stash/pull/423) ([tamalsaha](https://github.com/tamalsaha))
- Add install pkg for stash crds [\#422](https://github.com/appscode/stash/pull/422) ([tamalsaha](https://github.com/tamalsaha))
- Fix openapi spec for stash crds [\#421](https://github.com/appscode/stash/pull/421) ([tamalsaha](https://github.com/tamalsaha))
- Expose swagger.json [\#420](https://github.com/appscode/stash/pull/420) ([tamalsaha](https://github.com/tamalsaha))
- Show repository snapshot list [\#417](https://github.com/appscode/stash/pull/417) ([hossainemruz](https://github.com/hossainemruz))
- Add registry skeleton for snapshots [\#415](https://github.com/appscode/stash/pull/415) ([tamalsaha](https://github.com/tamalsaha))
- Update chart readme [\#413](https://github.com/appscode/stash/pull/413) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.3](https://github.com/appscode/stash/tree/0.7.0-rc.3) (2018-04-03)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.2...0.7.0-rc.3)

**Fixed bugs:**

- Use separate registry key for docker images [\#410](https://github.com/appscode/stash/pull/410) ([tamalsaha](https://github.com/tamalsaha))
- Revendor webhook util and jsonpatch fixes [\#400](https://github.com/appscode/stash/pull/400) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- hack/deploy/stash.sh: $? check does not work with set -e [\#403](https://github.com/appscode/stash/issues/403)

**Merged pull requests:**

- Add frontmatter for repository crd [\#412](https://github.com/appscode/stash/pull/412) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 0.7.0-rc.3 [\#411](https://github.com/appscode/stash/pull/411) ([tamalsaha](https://github.com/tamalsaha))
- Add test for recovery [\#409](https://github.com/appscode/stash/pull/409) ([hossainemruz](https://github.com/hossainemruz))
- Skip setting ListKind [\#407](https://github.com/appscode/stash/pull/407) ([tamalsaha](https://github.com/tamalsaha))
- Add CRD Validation [\#406](https://github.com/appscode/stash/pull/406) ([tamalsaha](https://github.com/tamalsaha))
- Generate openapi spec for stash api [\#405](https://github.com/appscode/stash/pull/405) ([tamalsaha](https://github.com/tamalsaha))
- Fix install script for minikube 0.24.x \(Kube 1.8.0\) [\#404](https://github.com/appscode/stash/pull/404) ([tamalsaha](https://github.com/tamalsaha))
- Skip downloading onessl if already installed [\#401](https://github.com/appscode/stash/pull/401) ([tamalsaha](https://github.com/tamalsaha))
- Use Restic spec hash instead of resource version to restart pods [\#399](https://github.com/appscode/stash/pull/399) ([tamalsaha](https://github.com/tamalsaha))
- Check for valid owner object [\#397](https://github.com/appscode/stash/pull/397) ([tamalsaha](https://github.com/tamalsaha))
- Create repository crd for each Restic repository [\#394](https://github.com/appscode/stash/pull/394) ([hossainemruz](https://github.com/hossainemruz))
- Revendor webhook library [\#393](https://github.com/appscode/stash/pull/393) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.2](https://github.com/appscode/stash/tree/0.7.0-rc.2) (2018-03-24)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.1...0.7.0-rc.2)

**Fixed bugs:**

- Fix --enable-analytics flag [\#387](https://github.com/appscode/stash/pull/387) ([tamalsaha](https://github.com/tamalsaha))
- Fix flag parsing in tests [\#386](https://github.com/appscode/stash/pull/386) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Prepare docs for 0.7.0-rc.2 [\#391](https://github.com/appscode/stash/pull/391) ([tamalsaha](https://github.com/tamalsaha))
- Add variable for dockerRegistry [\#390](https://github.com/appscode/stash/pull/390) ([tamalsaha](https://github.com/tamalsaha))
- Reorg objects deleted in uninstall command [\#389](https://github.com/appscode/stash/pull/389) ([tamalsaha](https://github.com/tamalsaha))
- Fix Statefulset Example [\#385](https://github.com/appscode/stash/pull/385) ([rzcastilho](https://github.com/rzcastilho))
- Rename --analytics to --enable-analytics [\#384](https://github.com/appscode/stash/pull/384) ([tamalsaha](https://github.com/tamalsaha))
- Use separated appscode/kubernetes-webhook-util package [\#383](https://github.com/appscode/stash/pull/383) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.1](https://github.com/appscode/stash/tree/0.7.0-rc.1) (2018-03-21)
[Full Changelog](https://github.com/appscode/stash/compare/0.7.0-rc.0...0.7.0-rc.1)

**Fixed bugs:**

- Don't enable mutator for StatefulSet updates [\#381](https://github.com/appscode/stash/pull/381) ([tamalsaha](https://github.com/tamalsaha))
- Stop using field selectors for CRDs [\#379](https://github.com/appscode/stash/pull/379) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- "DeprecatedServiceAccount not present in src" while converting unversioned StatefulSet to v1beta1.StatefulSet  [\#371](https://github.com/appscode/stash/issues/371)
- \[0.6.x\] Helm chart broken due to undocumented '--docker-registry' and other arguments [\#354](https://github.com/appscode/stash/issues/354)
- \[0.7.0-rc.0\] Fails on start-up with 'cluster doesn't provide requestheader-client-ca-file' [\#353](https://github.com/appscode/stash/issues/353)
- Ability to backup volumes with ReadWriteOnce access mode [\#350](https://github.com/appscode/stash/issues/350)
- Recovery not working! [\#303](https://github.com/appscode/stash/issues/303)

**Merged pull requests:**

- Update the image tag in operator.yaml [\#382](https://github.com/appscode/stash/pull/382) ([tamalsaha](https://github.com/tamalsaha))
- Update docs to 0.7.0-rc.1 [\#380](https://github.com/appscode/stash/pull/380) ([tamalsaha](https://github.com/tamalsaha))
-  Add types for Repository apigroup [\#377](https://github.com/appscode/stash/pull/377) ([tamalsaha](https://github.com/tamalsaha))
- Add missing front matter [\#376](https://github.com/appscode/stash/pull/376) ([tamalsaha](https://github.com/tamalsaha))
- Check for check job before creating it [\#375](https://github.com/appscode/stash/pull/375) ([galexrt](https://github.com/galexrt))
- Add travis.yaml [\#370](https://github.com/appscode/stash/pull/370) ([tamalsaha](https://github.com/tamalsaha))
- Add --purge flag [\#369](https://github.com/appscode/stash/pull/369) ([tamalsaha](https://github.com/tamalsaha))
- Make it clear that installer is a single command [\#365](https://github.com/appscode/stash/pull/365) ([tamalsaha](https://github.com/tamalsaha))
- Update installer [\#364](https://github.com/appscode/stash/pull/364) ([tamalsaha](https://github.com/tamalsaha))
- Replace initializers with mutation webhook for workloads [\#363](https://github.com/appscode/stash/pull/363) ([hossainemruz](https://github.com/hossainemruz))
- Update chart to match RBAC best practices for charts [\#362](https://github.com/appscode/stash/pull/362) ([tamalsaha](https://github.com/tamalsaha))
- Add checks to installer script [\#361](https://github.com/appscode/stash/pull/361) ([tamalsaha](https://github.com/tamalsaha))
- Use admission hook helpers from kutil [\#360](https://github.com/appscode/stash/pull/360) ([tamalsaha](https://github.com/tamalsaha))
- Fix admission webhook flag [\#359](https://github.com/appscode/stash/pull/359) ([tamalsaha](https://github.com/tamalsaha))
- Support --enable-admission-webhook=false [\#358](https://github.com/appscode/stash/pull/358) ([tamalsaha](https://github.com/tamalsaha))
- Support multiple webhooks of same apiversion [\#357](https://github.com/appscode/stash/pull/357) ([tamalsaha](https://github.com/tamalsaha))
- Sync chart to stable charts repo [\#356](https://github.com/appscode/stash/pull/356) ([tamalsaha](https://github.com/tamalsaha))
- Use restic 0.8.3 [\#355](https://github.com/appscode/stash/pull/355) ([tamalsaha](https://github.com/tamalsaha))
- Update README.md [\#352](https://github.com/appscode/stash/pull/352) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.0-rc.0](https://github.com/appscode/stash/tree/0.7.0-rc.0) (2018-02-20)
[Full Changelog](https://github.com/appscode/stash/compare/0.6.4...0.7.0-rc.0)

**Merged pull requests:**

- Document user roles [\#348](https://github.com/appscode/stash/pull/348) ([tamalsaha](https://github.com/tamalsaha))
- Add changelog for 0.7.0-rc.0 [\#347](https://github.com/appscode/stash/pull/347) ([tamalsaha](https://github.com/tamalsaha))
- Add a parameter to allow disabling initializers [\#346](https://github.com/appscode/stash/pull/346) ([mcanevet](https://github.com/mcanevet))
- Update readme to point to 0.6.4 [\#345](https://github.com/appscode/stash/pull/345) ([tamalsaha](https://github.com/tamalsaha))
- Implement offline backup for multiple replica [\#335](https://github.com/appscode/stash/pull/335) ([hossainemruz](https://github.com/hossainemruz))

## [0.6.4](https://github.com/appscode/stash/tree/0.6.4) (2018-02-20)
[Full Changelog](https://github.com/appscode/stash/compare/0.6.3...0.6.4)

**Fixed bugs:**

- Backup count rises even when backup/init fails [\#293](https://github.com/appscode/stash/issues/293)

**Closed issues:**

- Document HTTP endpoints [\#111](https://github.com/appscode/stash/issues/111)
- Support updating version of resitc side-car [\#72](https://github.com/appscode/stash/issues/72)

**Merged pull requests:**

- Update docs for 0.6.4 [\#344](https://github.com/appscode/stash/pull/344) ([tamalsaha](https://github.com/tamalsaha))
- Don't block deletion of owner by default [\#343](https://github.com/appscode/stash/pull/343) ([tamalsaha](https://github.com/tamalsaha))
- Don't block deletion of owner by default [\#342](https://github.com/appscode/stash/pull/342) ([tamalsaha](https://github.com/tamalsaha))
- Skip generating UpdateStatus method [\#341](https://github.com/appscode/stash/pull/341) ([tamalsaha](https://github.com/tamalsaha))
- Remove internal types [\#340](https://github.com/appscode/stash/pull/340) ([tamalsaha](https://github.com/tamalsaha))
- Use rbac/v1 apis [\#339](https://github.com/appscode/stash/pull/339) ([tamalsaha](https://github.com/tamalsaha))
- Add user roles [\#338](https://github.com/appscode/stash/pull/338) ([tamalsaha](https://github.com/tamalsaha))
- Use restic 0.8.2 [\#337](https://github.com/appscode/stash/pull/337) ([tamalsaha](https://github.com/tamalsaha))
- Use official code generator scripts [\#336](https://github.com/appscode/stash/pull/336) ([tamalsaha](https://github.com/tamalsaha))
- Update charts to support api registration [\#334](https://github.com/appscode/stash/pull/334) ([tamalsaha](https://github.com/tamalsaha))
- Fix e2e tests after webhook merger [\#333](https://github.com/appscode/stash/pull/333) ([tamalsaha](https://github.com/tamalsaha))
- Ensure stash can be run locally [\#332](https://github.com/appscode/stash/pull/332) ([tamalsaha](https://github.com/tamalsaha))
- Vendor client-go auth pkg [\#331](https://github.com/appscode/stash/pull/331) ([tamalsaha](https://github.com/tamalsaha))
- Update Grafana dashboard [\#330](https://github.com/appscode/stash/pull/330) ([galexrt](https://github.com/galexrt))
- Merge admission webhook and operator into one binary [\#329](https://github.com/appscode/stash/pull/329) ([tamalsaha](https://github.com/tamalsaha))
- Merge uninstall script into the stash.sh script [\#328](https://github.com/appscode/stash/pull/328) ([tamalsaha](https://github.com/tamalsaha))
- Implement informer factory for backup scheduler [\#325](https://github.com/appscode/stash/pull/325) ([hossainemruz](https://github.com/hossainemruz))
- Fixed abnormal pod recreation when Restic is deleted [\#322](https://github.com/appscode/stash/pull/322) ([hossainemruz](https://github.com/hossainemruz))
- Copy generic-admission-server into pkg [\#318](https://github.com/appscode/stash/pull/318) ([tamalsaha](https://github.com/tamalsaha))
- Use shared infromer factory [\#317](https://github.com/appscode/stash/pull/317) ([tamalsaha](https://github.com/tamalsaha))
- Use GetBaseVersion method from kutil [\#316](https://github.com/appscode/stash/pull/316) ([tamalsaha](https://github.com/tamalsaha))
- Implement Pause Restic [\#315](https://github.com/appscode/stash/pull/315) ([hossainemruz](https://github.com/hossainemruz))
- Fix webhook command description [\#314](https://github.com/appscode/stash/pull/314) ([tamalsaha](https://github.com/tamalsaha))
- Use rbac/v1beta1 api. [\#313](https://github.com/appscode/stash/pull/313) ([tamalsaha](https://github.com/tamalsaha))
- Support Create & Update operations in admission webhook [\#312](https://github.com/appscode/stash/pull/312) ([tamalsaha](https://github.com/tamalsaha))
- Merge webhook plugins into one. [\#311](https://github.com/appscode/stash/pull/311) ([tamalsaha](https://github.com/tamalsaha))
- Support private docker registry in installer [\#310](https://github.com/appscode/stash/pull/310) ([tamalsaha](https://github.com/tamalsaha))
- Compress go binaries [\#309](https://github.com/appscode/stash/pull/309) ([tamalsaha](https://github.com/tamalsaha))
- Rename --initializer flag to --enable-initializer [\#308](https://github.com/appscode/stash/pull/308) ([tamalsaha](https://github.com/tamalsaha))
- Remove STASH\_ROLE\_TYPE from installer scripts [\#307](https://github.com/appscode/stash/pull/307) ([tamalsaha](https://github.com/tamalsaha))
- Use rbac/v1 api [\#306](https://github.com/appscode/stash/pull/306) ([tamalsaha](https://github.com/tamalsaha))
- Use kubectl auth reconcile [\#305](https://github.com/appscode/stash/pull/305) ([tamalsaha](https://github.com/tamalsaha))
- Add --initializer flag to installer [\#304](https://github.com/appscode/stash/pull/304) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 0.7.0-alpha.0 [\#302](https://github.com/appscode/stash/pull/302) ([tamalsaha](https://github.com/tamalsaha))
- Change installer script [\#301](https://github.com/appscode/stash/pull/301) ([tamalsaha](https://github.com/tamalsaha))
- Added support for private docker registry [\#300](https://github.com/appscode/stash/pull/300) ([diptadas](https://github.com/diptadas))
- Add ValidatingAdmissionWebhook for Stash CRDs [\#299](https://github.com/appscode/stash/pull/299) ([tamalsaha](https://github.com/tamalsaha))
- Remove TPR to CRD migrator [\#298](https://github.com/appscode/stash/pull/298) ([tamalsaha](https://github.com/tamalsaha))
- Update dependencies to Kubernetes 1.9 [\#297](https://github.com/appscode/stash/pull/297) ([tamalsaha](https://github.com/tamalsaha))
- Write restic stderror in error events [\#296](https://github.com/appscode/stash/pull/296) ([diptadas](https://github.com/diptadas))
- Fixed backup count [\#295](https://github.com/appscode/stash/pull/295) ([diptadas](https://github.com/diptadas))
- Support self-signed ca cert for backends [\#294](https://github.com/appscode/stash/pull/294) ([hossainemruz](https://github.com/hossainemruz))

## [0.6.3](https://github.com/appscode/stash/tree/0.6.3) (2018-01-18)
[Full Changelog](https://github.com/appscode/stash/compare/0.6.2...0.6.3)

**Implemented enhancements:**

- Add Stash Backup Grafana dashboard to monitoring docs [\#285](https://github.com/appscode/stash/issues/285)
- Added Grafana Stash overview dashboard [\#286](https://github.com/appscode/stash/pull/286) ([galexrt](https://github.com/galexrt))

**Fixed bugs:**

- PushGateURL not given to sidecar container [\#283](https://github.com/appscode/stash/issues/283)
- Fix inline volumeSource marshalling for LocalSpec [\#289](https://github.com/appscode/stash/pull/289) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Test Failed: Invalid argument error in sidecar container [\#290](https://github.com/appscode/stash/issues/290)

**Merged pull requests:**

- Cleanup headless service [\#292](https://github.com/appscode/stash/pull/292) ([diptadas](https://github.com/diptadas))
- Fixed parsing argument error [\#291](https://github.com/appscode/stash/pull/291) ([diptadas](https://github.com/diptadas))
- Pass through logger flags [\#287](https://github.com/appscode/stash/pull/287) ([tamalsaha](https://github.com/tamalsaha))
- Pass --pushgateway-url for injected containers. [\#284](https://github.com/appscode/stash/pull/284) ([tamalsaha](https://github.com/tamalsaha))

## [0.6.2](https://github.com/appscode/stash/tree/0.6.2) (2018-01-05)
[Full Changelog](https://github.com/appscode/stash/compare/0.6.1...0.6.2)

**Fixed bugs:**

- Created stash-sidecar clusterrole is missing statefulsets permission [\#272](https://github.com/appscode/stash/issues/272)
- Garbage collect s/a and rolebindings for \*Jobs [\#271](https://github.com/appscode/stash/issues/271)
- Fix RBAC roles in chart [\#276](https://github.com/appscode/stash/pull/276) ([tamalsaha](https://github.com/tamalsaha))
- Garbage collect service-accounts and role-bindings for jobs [\#275](https://github.com/appscode/stash/pull/275) ([diptadas](https://github.com/diptadas))
- Fix new restic format in upgrade docs [\#274](https://github.com/appscode/stash/pull/274) ([tamalsaha](https://github.com/tamalsaha))
- Add statefulsets to stash-sidecar ClusterRole creation [\#273](https://github.com/appscode/stash/pull/273) ([galexrt](https://github.com/galexrt))

**Closed issues:**

- Image kubectl not found because of Kubernetes version [\#266](https://github.com/appscode/stash/issues/266)

**Merged pull requests:**

- Prepare docs for 0.6.2 release [\#278](https://github.com/appscode/stash/pull/278) ([tamalsaha](https://github.com/tamalsaha))
- Update Helm chart to use newer 'fullname' template that avoids duplicate \(e.g. 'stash-stash-...'\) resource names [\#277](https://github.com/appscode/stash/pull/277) ([whereisaaron](https://github.com/whereisaaron))
- Reduce operator permissions for service accounts [\#270](https://github.com/appscode/stash/pull/270) ([tamalsaha](https://github.com/tamalsaha))
- Fix formatting of uninstall.md [\#269](https://github.com/appscode/stash/pull/269) ([tamalsaha](https://github.com/tamalsaha))

## [0.6.1](https://github.com/appscode/stash/tree/0.6.1) (2018-01-03)
[Full Changelog](https://github.com/appscode/stash/compare/0.6.0...0.6.1)

**Fixed bugs:**

- Error while running restic [\#256](https://github.com/appscode/stash/issues/256)

**Closed issues:**

- Unable to use non-aws S3 backend [\#226](https://github.com/appscode/stash/issues/226)

**Merged pull requests:**

- Prepare docs for 0.6.1 [\#268](https://github.com/appscode/stash/pull/268) ([tamalsaha](https://github.com/tamalsaha))

## [0.6.0](https://github.com/appscode/stash/tree/0.6.0) (2018-01-03)
[Full Changelog](https://github.com/appscode/stash/compare/0.4.2...0.6.0)

**Implemented enhancements:**

- Feature: Support offline consistent backups [\#225](https://github.com/appscode/stash/issues/225)
- Collect ideas on how to improve recovery process [\#131](https://github.com/appscode/stash/issues/131)
- Use log.LEVEL\(\) instead of fmt.Printf\(\) [\#252](https://github.com/appscode/stash/pull/252) ([galexrt](https://github.com/galexrt))

**Fixed bugs:**

- Fix ConfigMap Name in Leader Election [\#227](https://github.com/appscode/stash/issues/227)
- StatefulSet: Forbidden: pod updates may not add or remove containers [\#191](https://github.com/appscode/stash/issues/191)
- Events are not recording for Recovery [\#219](https://github.com/appscode/stash/issues/219)
- \[0.5.0\] Record backup event on kubernetes failure [\#212](https://github.com/appscode/stash/issues/212)
- Fix kubectl version parsing generation in GKE [\#267](https://github.com/appscode/stash/pull/267) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Replace fmt.Print\* with log statements [\#248](https://github.com/appscode/stash/issues/248)
- Dynamically create stash-sidecar ClusterRole in operator [\#220](https://github.com/appscode/stash/issues/220)
- LeaderElection part -2  [\#218](https://github.com/appscode/stash/issues/218)
- Reimplement CheckRecoveryJob using Job watcher [\#216](https://github.com/appscode/stash/issues/216)
- Enable --cache-dir [\#238](https://github.com/appscode/stash/issues/238)
- Upgrade procedure for 0.5.1 -\> 0.6.0 [\#237](https://github.com/appscode/stash/issues/237)
- Test RBAC setup [\#224](https://github.com/appscode/stash/issues/224)
- Record recovery status for individual FileGroup [\#213](https://github.com/appscode/stash/issues/213)
- Periodically run restic check [\#195](https://github.com/appscode/stash/issues/195)
- Handle Deployment etc with replicas \> 1 [\#140](https://github.com/appscode/stash/issues/140)
- Support Backblaze B2 as backend [\#125](https://github.com/appscode/stash/issues/125)
- Turn Stash operator into an Initializer [\#5](https://github.com/appscode/stash/issues/5)

**Merged pull requests:**

- Detect analytics client id using env vars [\#265](https://github.com/appscode/stash/pull/265) ([tamalsaha](https://github.com/tamalsaha))
- Repare docs for 0.6.0 release [\#264](https://github.com/appscode/stash/pull/264) ([tamalsaha](https://github.com/tamalsaha))
- Reorganize docs [\#263](https://github.com/appscode/stash/pull/263) ([tamalsaha](https://github.com/tamalsaha))
- Add support for B2 [\#262](https://github.com/appscode/stash/pull/262) ([tamalsaha](https://github.com/tamalsaha))
- Update restic website link [\#261](https://github.com/appscode/stash/pull/261) ([tamalsaha](https://github.com/tamalsaha))
- Update docs for unified LocalSpec [\#260](https://github.com/appscode/stash/pull/260) ([diptadas](https://github.com/diptadas))
- Unify LocalSpec and RecoveredVolume [\#259](https://github.com/appscode/stash/pull/259) ([diptadas](https://github.com/diptadas))
- Remove restic-dependency from recovery [\#258](https://github.com/appscode/stash/pull/258) ([diptadas](https://github.com/diptadas))
- Update restic version to 0.8.1 [\#257](https://github.com/appscode/stash/pull/257) ([tamalsaha](https://github.com/tamalsaha))
- Use cmp methods from kutil [\#255](https://github.com/appscode/stash/pull/255) ([tamalsaha](https://github.com/tamalsaha))
- Remove TryPatch methods [\#254](https://github.com/appscode/stash/pull/254) ([tamalsaha](https://github.com/tamalsaha))
- Log operator version on start [\#253](https://github.com/appscode/stash/pull/253) ([galexrt](https://github.com/galexrt))
- Use verb type for mutation [\#251](https://github.com/appscode/stash/pull/251) ([tamalsaha](https://github.com/tamalsaha))
- Use CreateOrPatchCronJob from kutil [\#250](https://github.com/appscode/stash/pull/250) ([tamalsaha](https://github.com/tamalsaha))
- Indicate mutation in PATCH helper method return [\#249](https://github.com/appscode/stash/pull/249) ([tamalsaha](https://github.com/tamalsaha))
- Simplify clientID generation for analytics [\#247](https://github.com/appscode/stash/pull/247) ([tamalsaha](https://github.com/tamalsaha))
- Set analytics clientID [\#246](https://github.com/appscode/stash/pull/246) ([tamalsaha](https://github.com/tamalsaha))
- Reorganize docs [\#245](https://github.com/appscode/stash/pull/245) ([tamalsaha](https://github.com/tamalsaha))
- Upgrade procedure for 0.5.1 to 0.6.0 [\#243](https://github.com/appscode/stash/pull/243) ([diptadas](https://github.com/diptadas))
- Fix retentionPolicyName not found error [\#242](https://github.com/appscode/stash/pull/242) ([diptadas](https://github.com/diptadas))
- Enable Restic cahce-dir flag [\#241](https://github.com/appscode/stash/pull/241) ([diptadas](https://github.com/diptadas))
- Use lower case workload.kind in prefix [\#240](https://github.com/appscode/stash/pull/240) ([diptadas](https://github.com/diptadas))
- Use RegisterCRDs helper [\#239](https://github.com/appscode/stash/pull/239) ([tamalsaha](https://github.com/tamalsaha))
- Update docs [\#236](https://github.com/appscode/stash/pull/236) ([diptadas](https://github.com/diptadas))
- Change left\_menu -\> menu\_name [\#235](https://github.com/appscode/stash/pull/235) ([sajibcse68](https://github.com/sajibcse68))
- Revendor dependencies [\#234](https://github.com/appscode/stash/pull/234) ([tamalsaha](https://github.com/tamalsaha))
- Add aliases for README file in front matter [\#233](https://github.com/appscode/stash/pull/233) ([sajibcse68](https://github.com/sajibcse68))
- Update bundles restic to 0.8.0 [\#232](https://github.com/appscode/stash/pull/232) ([tamalsaha](https://github.com/tamalsaha))
- Add Docs Front Matter for 0.5.1 [\#231](https://github.com/appscode/stash/pull/231) ([sajibcse68](https://github.com/sajibcse68))
- Revendor kutil [\#230](https://github.com/appscode/stash/pull/230) ([tamalsaha](https://github.com/tamalsaha))
- Implement offline backup [\#229](https://github.com/appscode/stash/pull/229) ([diptadas](https://github.com/diptadas))
- Fix Configmap Name in Leader Election [\#228](https://github.com/appscode/stash/pull/228) ([diptadas](https://github.com/diptadas))
- Run `restic check` once every 3 days [\#223](https://github.com/appscode/stash/pull/223) ([tamalsaha](https://github.com/tamalsaha))
- Record recovery status for individual FileGroup [\#222](https://github.com/appscode/stash/pull/222) ([tamalsaha](https://github.com/tamalsaha))
- Dynamically create stash-sidecar ClusterRole in operator [\#221](https://github.com/appscode/stash/pull/221) ([tamalsaha](https://github.com/tamalsaha))
- Make stash chart namespaced [\#210](https://github.com/appscode/stash/pull/210) ([tamalsaha](https://github.com/tamalsaha))
- Implement workload initializer in stash operator [\#207](https://github.com/appscode/stash/pull/207) ([diptadas](https://github.com/diptadas))
- Leader election for deployment, replica set and rc [\#206](https://github.com/appscode/stash/pull/206) ([diptadas](https://github.com/diptadas))
- Revise RetentionPolicy in Restic Api [\#205](https://github.com/appscode/stash/pull/205) ([diptadas](https://github.com/diptadas))
- Implement Recovery for Restic Backup [\#202](https://github.com/appscode/stash/pull/202) ([diptadas](https://github.com/diptadas))

## [0.4.2](https://github.com/appscode/stash/tree/0.4.2) (2017-11-03)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.1...0.4.2)

**Merged pull requests:**

- Upgrade restic binary to 0.7.3 [\#209](https://github.com/appscode/stash/pull/209) ([tamalsaha](https://github.com/tamalsaha))
- Fix RBAC permission for release 0.4 [\#208](https://github.com/appscode/stash/pull/208) ([tamalsaha](https://github.com/tamalsaha))
- Change `k8s.io/api/core/v1` pkg alias to core [\#204](https://github.com/appscode/stash/pull/204) ([tamalsaha](https://github.com/tamalsaha))
- Use client-go 5.0 [\#203](https://github.com/appscode/stash/pull/203) ([tamalsaha](https://github.com/tamalsaha))
- Add recovery CRD [\#201](https://github.com/appscode/stash/pull/201) ([diptadas](https://github.com/diptadas))

## [0.5.1](https://github.com/appscode/stash/tree/0.5.1) (2017-10-10)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.0...0.5.1)

**Fixed bugs:**

- invalid header field value for key Authorization - DO s3 bucket [\#189](https://github.com/appscode/stash/issues/189)
- Kops + AWS: cannot unmarshal array into Go value of type types.ContainerJSON [\#147](https://github.com/appscode/stash/issues/147)

**Closed issues:**

- Cut a new release with restic 0.7.1 [\#145](https://github.com/appscode/stash/issues/145)
- Use fixed Hostname for ReplicaSet etc [\#165](https://github.com/appscode/stash/issues/165)
- Update docs for restic tags [\#143](https://github.com/appscode/stash/issues/143)
- Document how to use with kubectl [\#142](https://github.com/appscode/stash/issues/142)

**Merged pull requests:**

- Correctly detect "default" service account [\#200](https://github.com/appscode/stash/pull/200) ([tamalsaha](https://github.com/tamalsaha))
- Clarify that --tag foo,tag bar style tags are not supported. [\#199](https://github.com/appscode/stash/pull/199) ([tamalsaha](https://github.com/tamalsaha))
- Set hostname based on resource type [\#198](https://github.com/appscode/stash/pull/198) ([tamalsaha](https://github.com/tamalsaha))
- Document how to detect operator version [\#196](https://github.com/appscode/stash/pull/196) ([tamalsaha](https://github.com/tamalsaha))
- Manage RoleBinding for rbac enabled cluster [\#197](https://github.com/appscode/stash/pull/197) ([tamalsaha](https://github.com/tamalsaha))

## [0.5.0](https://github.com/appscode/stash/tree/0.5.0) (2017-10-10)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.0-beta.3...0.5.0)

**Closed issues:**

- Apply restic.appscode.com/config annotations on Pod templates [\#141](https://github.com/appscode/stash/issues/141)

**Merged pull requests:**

- Revendor forked robfig/cron [\#139](https://github.com/appscode/stash/pull/139) ([tamalsaha](https://github.com/tamalsaha))

## [0.5.0-beta.3](https://github.com/appscode/stash/tree/0.5.0-beta.3) (2017-10-10)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.0-beta.2...0.5.0-beta.3)

**Merged pull requests:**

- Use workqueue for scheduler [\#194](https://github.com/appscode/stash/pull/194) ([tamalsaha](https://github.com/tamalsaha))

## [0.5.0-beta.2](https://github.com/appscode/stash/tree/0.5.0-beta.2) (2017-10-09)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.0-beta.1...0.5.0-beta.2)

**Merged pull requests:**

- Add tests for DO [\#193](https://github.com/appscode/stash/pull/193) ([tamalsaha](https://github.com/tamalsaha))
- Update tutorial [\#186](https://github.com/appscode/stash/pull/186) ([diptadas](https://github.com/diptadas))

## [0.5.0-beta.1](https://github.com/appscode/stash/tree/0.5.0-beta.1) (2017-10-09)
[Full Changelog](https://github.com/appscode/stash/compare/0.5.0-beta.0...0.5.0-beta.1)

**Fixed bugs:**

- \[Bug\] Success/Fail prometheus metrics inverted condition [\#175](https://github.com/appscode/stash/issues/175)

**Closed issues:**

- `should backup new DaemonSet` failed [\#155](https://github.com/appscode/stash/issues/155)
- Use DaemonSet update \(1.6\) [\#154](https://github.com/appscode/stash/issues/154)

**Merged pull requests:**

- Fix prometheus metrics collection [\#192](https://github.com/appscode/stash/pull/192) ([tamalsaha](https://github.com/tamalsaha))
- Fix StatefulSet tests [\#190](https://github.com/appscode/stash/pull/190) ([tamalsaha](https://github.com/tamalsaha))
- Replace reflect.Equal with github.com/google/go-cmp [\#188](https://github.com/appscode/stash/pull/188) ([tamalsaha](https://github.com/tamalsaha))
- Skip ReplicaSet owned by Deployments [\#187](https://github.com/appscode/stash/pull/187) ([tamalsaha](https://github.com/tamalsaha))

## [0.5.0-beta.0](https://github.com/appscode/stash/tree/0.5.0-beta.0) (2017-10-09)
[Full Changelog](https://github.com/appscode/stash/compare/0.4.1...0.5.0-beta.0)

**Implemented enhancements:**

- Migrate TPR to CRD [\#160](https://github.com/appscode/stash/pull/160) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Error in request: v1.ListOptions is not suitable for converting to "v1" [\#153](https://github.com/appscode/stash/issues/153)
- Fix client-go updates [\#159](https://github.com/appscode/stash/pull/159) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Switch to CustomResourceDefinitions [\#97](https://github.com/appscode/stash/issues/97)
- Use client-go 4.0.0 [\#56](https://github.com/appscode/stash/issues/56)

**Merged pull requests:**

- Prepare docs for 5.0.0-beta.0 [\#185](https://github.com/appscode/stash/pull/185) ([tamalsaha](https://github.com/tamalsaha))
- Set namespaceIndex as indexer [\#184](https://github.com/appscode/stash/pull/184) ([tamalsaha](https://github.com/tamalsaha))
- Fix e2e tests [\#183](https://github.com/appscode/stash/pull/183) ([tamalsaha](https://github.com/tamalsaha))
- Use workqueue [\#182](https://github.com/appscode/stash/pull/182) ([tamalsaha](https://github.com/tamalsaha))
- Use Deployment from apps/v1beta1 [\#181](https://github.com/appscode/stash/pull/181) ([tamalsaha](https://github.com/tamalsaha))
- Delete \*.generated.go files for ugorji [\#180](https://github.com/appscode/stash/pull/180) ([tamalsaha](https://github.com/tamalsaha))
- Use WaitForCRDReady from kutil [\#179](https://github.com/appscode/stash/pull/179) ([tamalsaha](https://github.com/tamalsaha))
- Only watch apps/v1beta1 Deployment [\#178](https://github.com/appscode/stash/pull/178) ([tamalsaha](https://github.com/tamalsaha))
- Move kutil to client package [\#177](https://github.com/appscode/stash/pull/177) ([tamalsaha](https://github.com/tamalsaha))
- Generate ugorji stuff [\#176](https://github.com/appscode/stash/pull/176) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 0.5.0 [\#174](https://github.com/appscode/stash/pull/174) ([tamalsaha](https://github.com/tamalsaha))
- Install stash as a critical addon [\#173](https://github.com/appscode/stash/pull/173) ([tamalsaha](https://github.com/tamalsaha))
- Set RESTIC\_VER to 0.7.3 [\#172](https://github.com/appscode/stash/pull/172) ([tamalsaha](https://github.com/tamalsaha))
- Refresh charts to match recent convention [\#171](https://github.com/appscode/stash/pull/171) ([tamalsaha](https://github.com/tamalsaha))
- Update kutil [\#170](https://github.com/appscode/stash/pull/170) ([tamalsaha](https://github.com/tamalsaha))
- Fix deployment name in tutorial [\#169](https://github.com/appscode/stash/pull/169) ([the-redback](https://github.com/the-redback))
- Fix command in Developer-guide [\#168](https://github.com/appscode/stash/pull/168) ([the-redback](https://github.com/the-redback))
- Use apis/v1alpha1 instead of internal version [\#167](https://github.com/appscode/stash/pull/167) ([tamalsaha](https://github.com/tamalsaha))
- Remove resource:path [\#166](https://github.com/appscode/stash/pull/166) ([tamalsaha](https://github.com/tamalsaha))
- Move analytics collector to root command [\#164](https://github.com/appscode/stash/pull/164) ([tamalsaha](https://github.com/tamalsaha))
- Use kubernetes/code-generator [\#163](https://github.com/appscode/stash/pull/163) ([tamalsaha](https://github.com/tamalsaha))
- Revendor k8s.io/apiextensions-apiserver [\#162](https://github.com/appscode/stash/pull/162) ([tamalsaha](https://github.com/tamalsaha))
- Update kutil dependency [\#158](https://github.com/appscode/stash/pull/158) ([tamalsaha](https://github.com/tamalsaha))
- Use CheckAPIVersion\(\) [\#157](https://github.com/appscode/stash/pull/157) ([tamalsaha](https://github.com/tamalsaha))
- Use PATCH api instead of UPDATE [\#156](https://github.com/appscode/stash/pull/156) ([tamalsaha](https://github.com/tamalsaha))
- Check version using semver library [\#152](https://github.com/appscode/stash/pull/152) ([tamalsaha](https://github.com/tamalsaha))
- Support adding Sidecar containers for StatefulSet. [\#151](https://github.com/appscode/stash/pull/151) ([tamalsaha](https://github.com/tamalsaha))
- Update client-go to 4.0.0 [\#150](https://github.com/appscode/stash/pull/150) ([tamalsaha](https://github.com/tamalsaha))
- Update build commands for restic. [\#149](https://github.com/appscode/stash/pull/149) ([tamalsaha](https://github.com/tamalsaha))
- Update client-go to 3.0.0 from 3.0.0-beta [\#148](https://github.com/appscode/stash/pull/148) ([tamalsaha](https://github.com/tamalsaha))
- Add uninstall.sh script [\#144](https://github.com/appscode/stash/pull/144) ([tamalsaha](https://github.com/tamalsaha))
- Fix typos of tutorial.md file [\#138](https://github.com/appscode/stash/pull/138) ([sajibcse68](https://github.com/sajibcse68))

## [0.4.1](https://github.com/appscode/stash/tree/0.4.1) (2017-07-19)
[Full Changelog](https://github.com/appscode/stash/compare/0.4.0...0.4.1)

**Fixed bugs:**

- Fix Fake restic resource Url [\#137](https://github.com/appscode/stash/pull/137) ([sadlil](https://github.com/sadlil))

## [0.4.0](https://github.com/appscode/stash/tree/0.4.0) (2017-07-07)
[Full Changelog](https://github.com/appscode/stash/compare/0.3.1...0.4.0)

**Closed issues:**

- Use osm as the bucket manipulator [\#3](https://github.com/appscode/stash/issues/3)
- Update restic [\#133](https://github.com/appscode/stash/issues/133)
- Document required RBAC permissions [\#123](https://github.com/appscode/stash/issues/123)

**Merged pull requests:**

- Rename RepositorySecretName to StorageSecretName [\#135](https://github.com/appscode/stash/pull/135) ([tamalsaha](https://github.com/tamalsaha))
- Rename Volume to VolumeSource [\#134](https://github.com/appscode/stash/pull/134) ([tamalsaha](https://github.com/tamalsaha))
- Use VolumeSource instead of Volume for Local backend. [\#132](https://github.com/appscode/stash/pull/132) ([tamalsaha](https://github.com/tamalsaha))

## [0.3.1](https://github.com/appscode/stash/tree/0.3.1) (2017-07-04)
[Full Changelog](https://github.com/appscode/stash/compare/0.3.0...0.3.1)

**Merged pull requests:**

- Add tests for swift [\#130](https://github.com/appscode/stash/pull/130) ([tamalsaha](https://github.com/tamalsaha))

## [0.3.0](https://github.com/appscode/stash/tree/0.3.0) (2017-07-04)
[Full Changelog](https://github.com/appscode/stash/compare/0.2.0...0.3.0)

**Fixed bugs:**

- Fix GCS [\#122](https://github.com/appscode/stash/issues/122)

**Closed issues:**

- Support resource [\#128](https://github.com/appscode/stash/issues/128)
- Document FindRestic will match first one [\#119](https://github.com/appscode/stash/issues/119)
- Document e2e test setup process. [\#108](https://github.com/appscode/stash/issues/108)
- Fix charts [\#87](https://github.com/appscode/stash/issues/87)

**Merged pull requests:**

- Support setting compute resources for sidecar [\#129](https://github.com/appscode/stash/pull/129) ([tamalsaha](https://github.com/tamalsaha))
- Fix RBAC docs [\#127](https://github.com/appscode/stash/pull/127) ([tamalsaha](https://github.com/tamalsaha))
- Document swift [\#124](https://github.com/appscode/stash/pull/124) ([tamalsaha](https://github.com/tamalsaha))

## [0.2.0](https://github.com/appscode/stash/tree/0.2.0) (2017-06-30)
[Full Changelog](https://github.com/appscode/stash/compare/0.1.0...0.2.0)

**Implemented enhancements:**

- Don't run forget if missing retention policy [\#100](https://github.com/appscode/stash/issues/100)
- Move prefix-hostname to Restic tpr [\#96](https://github.com/appscode/stash/issues/96)

**Fixed bugs:**

- Mount source volume [\#112](https://github.com/appscode/stash/issues/112)
- Test restic URL is generated correctly when optional parts are missing [\#98](https://github.com/appscode/stash/issues/98)
- Handle updated restic selectors [\#95](https://github.com/appscode/stash/issues/95)

**Closed issues:**

- Link to sidecar flags. [\#109](https://github.com/appscode/stash/issues/109)
- Link back to tutorial from docs pages. [\#107](https://github.com/appscode/stash/issues/107)
- Document various implications of Restic update [\#103](https://github.com/appscode/stash/issues/103)
- Add retention policy options [\#101](https://github.com/appscode/stash/issues/101)
- Handle updating local backend. [\#105](https://github.com/appscode/stash/issues/105)
- Set Temp dir ENV var [\#102](https://github.com/appscode/stash/issues/102)
- Cleanup documentation [\#86](https://github.com/appscode/stash/issues/86)
- Updating Local backend does not update pods. [\#71](https://github.com/appscode/stash/issues/71)

**Merged pull requests:**

- Part 6 - Update docs [\#121](https://github.com/appscode/stash/pull/121) ([tamalsaha](https://github.com/tamalsaha))
- Update docs [\#120](https://github.com/appscode/stash/pull/120) ([tamalsaha](https://github.com/tamalsaha))
- Various bug fixes [\#118](https://github.com/appscode/stash/pull/118) ([tamalsaha](https://github.com/tamalsaha))
- Update pitch [\#117](https://github.com/appscode/stash/pull/117) ([tamalsaha](https://github.com/tamalsaha))
- Part 5 - User Guide [\#114](https://github.com/appscode/stash/pull/114) ([tamalsaha](https://github.com/tamalsaha))
- Part 4- User Guide [\#113](https://github.com/appscode/stash/pull/113) ([tamalsaha](https://github.com/tamalsaha))
- Part 3 - User Guide [\#110](https://github.com/appscode/stash/pull/110) ([tamalsaha](https://github.com/tamalsaha))
- Update user guide [\#94](https://github.com/appscode/stash/pull/94) ([tamalsaha](https://github.com/tamalsaha))
- Create separate restic for each type of backend. [\#92](https://github.com/appscode/stash/pull/92) ([tamalsaha](https://github.com/tamalsaha))
- Remove selectors so that `template.metadata.labels` are used [\#91](https://github.com/appscode/stash/pull/91) ([tamalsaha](https://github.com/tamalsaha))
- Update Stash chart [\#89](https://github.com/appscode/stash/pull/89) ([tamalsaha](https://github.com/tamalsaha))
- Various changes to RetentionPolicy	 [\#116](https://github.com/appscode/stash/pull/116) ([tamalsaha](https://github.com/tamalsaha))
- Set TMPDIR env var for restic [\#115](https://github.com/appscode/stash/pull/115) ([tamalsaha](https://github.com/tamalsaha))
- Part - 2 of User guide [\#99](https://github.com/appscode/stash/pull/99) ([tamalsaha](https://github.com/tamalsaha))
- Update Prometheus job name to use restic ns & name [\#93](https://github.com/appscode/stash/pull/93) ([tamalsaha](https://github.com/tamalsaha))
- Add docs for commands [\#90](https://github.com/appscode/stash/pull/90) ([tamalsaha](https://github.com/tamalsaha))
- Fix dev guide [\#88](https://github.com/appscode/stash/pull/88) ([tamalsaha](https://github.com/tamalsaha))

## [0.1.0](https://github.com/appscode/stash/tree/0.1.0) (2017-06-27)
**Implemented enhancements:**

- Allow modifying the cron expression [\#21](https://github.com/appscode/stash/issues/21)
- Use RBAC objects for operator. [\#64](https://github.com/appscode/stash/issues/64)
- Support Azure as backup destination [\#35](https://github.com/appscode/stash/issues/35)
- Support GCS as backup destination [\#34](https://github.com/appscode/stash/issues/34)
- Change Destination definition to point to S3 [\#33](https://github.com/appscode/stash/issues/33)
- TODOs [\#22](https://github.com/appscode/stash/issues/22)
- Send performance stats to Prometheus [\#9](https://github.com/appscode/stash/issues/9)

**Fixed bugs:**

- Bubble up errors to the caller. [\#24](https://github.com/appscode/stash/issues/24)
- Fix registration of wrong group [\#39](https://github.com/appscode/stash/pull/39) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Add /snapshots endpoint in operator [\#81](https://github.com/appscode/stash/issues/81)
- CLI: restic-ctl [\#8](https://github.com/appscode/stash/issues/8)
- Sanitize metric labels [\#68](https://github.com/appscode/stash/issues/68)
- Mount an empty directory to write local files. [\#61](https://github.com/appscode/stash/issues/61)
- Support BackBlaze as backup destination [\#60](https://github.com/appscode/stash/issues/60)
- Support Swift as backup destination [\#59](https://github.com/appscode/stash/issues/59)
- Add e2e tests using Ginkgo [\#57](https://github.com/appscode/stash/issues/57)
- Review analytics [\#55](https://github.com/appscode/stash/issues/55)
- Support updated Kube object versions [\#42](https://github.com/appscode/stash/issues/42)
- Update restic to 0.6.x [\#32](https://github.com/appscode/stash/issues/32)
- Add analytics [\#31](https://github.com/appscode/stash/issues/31)
- HTTP api to exposing restic repository data [\#7](https://github.com/appscode/stash/issues/7)
- Provision new restic repositories [\#6](https://github.com/appscode/stash/issues/6)
- Proposal: Imeplement Restic TPR Resource for Kubernetes [\#1](https://github.com/appscode/stash/issues/1)

**Merged pull requests:**

- Add e2e tests for major cloud providers [\#84](https://github.com/appscode/stash/pull/84) ([tamalsaha](https://github.com/tamalsaha))
- Add /snapshots endpoint in operator [\#82](https://github.com/appscode/stash/pull/82) ([tamalsaha](https://github.com/tamalsaha))
- Handle update conflicts [\#78](https://github.com/appscode/stash/pull/78) ([tamalsaha](https://github.com/tamalsaha))
- Test e2e tests [\#76](https://github.com/appscode/stash/pull/76) ([tamalsaha](https://github.com/tamalsaha))
- Delete old testify tests [\#75](https://github.com/appscode/stash/pull/75) ([tamalsaha](https://github.com/tamalsaha))
- Create a cli wrapper for restic [\#74](https://github.com/appscode/stash/pull/74) ([tamalsaha](https://github.com/tamalsaha))
- Revise EnsureXXXSidecar methods [\#73](https://github.com/appscode/stash/pull/73) ([tamalsaha](https://github.com/tamalsaha))
- Add ginkgo based e2e tests [\#70](https://github.com/appscode/stash/pull/70) ([tamalsaha](https://github.com/tamalsaha))
- Push metrics to Prometheus push gateway [\#67](https://github.com/appscode/stash/pull/67) ([tamalsaha](https://github.com/tamalsaha))
- Use go-sh to execute restic commands [\#63](https://github.com/appscode/stash/pull/63) ([tamalsaha](https://github.com/tamalsaha))
- Add scratchDir & prefixHostname flags [\#62](https://github.com/appscode/stash/pull/62) ([tamalsaha](https://github.com/tamalsaha))
- Support remote backends [\#58](https://github.com/appscode/stash/pull/58) ([tamalsaha](https://github.com/tamalsaha))
- Organize backup code. [\#54](https://github.com/appscode/stash/pull/54) ([tamalsaha](https://github.com/tamalsaha))
- Synchronize scheduler reconfiguration [\#53](https://github.com/appscode/stash/pull/53) ([tamalsaha](https://github.com/tamalsaha))
- Fix unit tests [\#51](https://github.com/appscode/stash/pull/51) ([tamalsaha](https://github.com/tamalsaha))
- Check docker image tag before starting operator [\#45](https://github.com/appscode/stash/pull/45) ([tamalsaha](https://github.com/tamalsaha))
- Expose metrics from operator [\#44](https://github.com/appscode/stash/pull/44) ([tamalsaha](https://github.com/tamalsaha))
- Add analytics [\#41](https://github.com/appscode/stash/pull/41) ([aerokite](https://github.com/aerokite))
- Use V1alpha1SchemeGroupVersion for Restik [\#40](https://github.com/appscode/stash/pull/40) ([aerokite](https://github.com/aerokite))
- Fix status update [\#38](https://github.com/appscode/stash/pull/38) ([saumanbiswas](https://github.com/saumanbiswas))
- Upgrade restic version to 0.6.1 [\#37](https://github.com/appscode/stash/pull/37) ([tamalsaha](https://github.com/tamalsaha))
- Change api version to v1alpha1 [\#30](https://github.com/appscode/stash/pull/30) ([tamalsaha](https://github.com/tamalsaha))
- Rename function and structure [\#29](https://github.com/appscode/stash/pull/29) ([saumanbiswas](https://github.com/saumanbiswas))
- Rename Backup into Restik [\#28](https://github.com/appscode/stash/pull/28) ([saumanbiswas](https://github.com/saumanbiswas))
- Move api from k8s-addons [\#27](https://github.com/appscode/stash/pull/27) ([saumanbiswas](https://github.com/saumanbiswas))
- Bubble up errors to caller [\#26](https://github.com/appscode/stash/pull/26) ([saumanbiswas](https://github.com/saumanbiswas))
- Allow modifying the cron expression [\#25](https://github.com/appscode/stash/pull/25) ([saumanbiswas](https://github.com/saumanbiswas))
- Use unversioned time [\#23](https://github.com/appscode/stash/pull/23) ([tamalsaha](https://github.com/tamalsaha))
- Restik chart [\#20](https://github.com/appscode/stash/pull/20) ([saumanbiswas](https://github.com/saumanbiswas))
- example added [\#19](https://github.com/appscode/stash/pull/19) ([saumanbiswas](https://github.com/saumanbiswas))
- Move restik api and client to k8s-addons [\#18](https://github.com/appscode/stash/pull/18) ([saumanbiswas](https://github.com/saumanbiswas))
- Error print fix [\#17](https://github.com/appscode/stash/pull/17) ([saumanbiswas](https://github.com/saumanbiswas))
- Check group registration [\#16](https://github.com/appscode/stash/pull/16) ([saumanbiswas](https://github.com/saumanbiswas))
- Restik docs [\#15](https://github.com/appscode/stash/pull/15) ([saumanbiswas](https://github.com/saumanbiswas))
- Restik unit test, e2e test [\#14](https://github.com/appscode/stash/pull/14) ([saumanbiswas](https://github.com/saumanbiswas))
- Restik create delete initial implementation [\#12](https://github.com/appscode/stash/pull/12) ([saumanbiswas](https://github.com/saumanbiswas))
- Build docker image [\#11](https://github.com/appscode/stash/pull/11) ([tamalsaha](https://github.com/tamalsaha))
- Clone skeleton from appscode/k3pc [\#10](https://github.com/appscode/stash/pull/10) ([tamalsaha](https://github.com/tamalsaha))
- Fix e2e tests [\#83](https://github.com/appscode/stash/pull/83) ([tamalsaha](https://github.com/tamalsaha))
- Mount scratchDir with operator [\#80](https://github.com/appscode/stash/pull/80) ([tamalsaha](https://github.com/tamalsaha))
- Fix scheduler  [\#79](https://github.com/appscode/stash/pull/79) ([tamalsaha](https://github.com/tamalsaha))
- Create RBAC objects for operator [\#69](https://github.com/appscode/stash/pull/69) ([tamalsaha](https://github.com/tamalsaha))
- Mount labels using Downward api [\#66](https://github.com/appscode/stash/pull/66) ([tamalsaha](https://github.com/tamalsaha))
- Vendor go-sh dependency [\#65](https://github.com/appscode/stash/pull/65) ([tamalsaha](https://github.com/tamalsaha))
- Update e2e tests [\#52](https://github.com/appscode/stash/pull/52) ([tamalsaha](https://github.com/tamalsaha))
- Run watchers for preferred api group version kind [\#50](https://github.com/appscode/stash/pull/50) ([tamalsaha](https://github.com/tamalsaha))
- Build restic from source by default [\#49](https://github.com/appscode/stash/pull/49) ([tamalsaha](https://github.com/tamalsaha))
- Watch individual object types. [\#48](https://github.com/appscode/stash/pull/48) ([tamalsaha](https://github.com/tamalsaha))
- Various code cleanup [\#47](https://github.com/appscode/stash/pull/47) ([tamalsaha](https://github.com/tamalsaha))
- Reorganize cron controller [\#46](https://github.com/appscode/stash/pull/46) ([tamalsaha](https://github.com/tamalsaha))
- Run push gateway as a side-car for restik operator. [\#43](https://github.com/appscode/stash/pull/43) ([tamalsaha](https://github.com/tamalsaha))
- Use client-go [\#36](https://github.com/appscode/stash/pull/36) ([tamalsaha](https://github.com/tamalsaha))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*