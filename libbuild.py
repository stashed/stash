#!/usr/bin/env python

# Needed for antipackage with python 2
from __future__ import absolute_import

import datetime
import fnmatch
import glob
import io
import json
import os
import os.path
import random
import re
import socket
import string
import subprocess
import sys
import yaml
from collections import Counter, OrderedDict
from os.path import expandvars

REPO_ROOT = ''
BIN_MATRIX = None
BUCKET_MATRIX = None
ENV = os.getenv('APPSCODE_ENV', 'dev').lower()


def _goenv():
    env = {}
    for line in subprocess.check_output(['go', 'env']).split('\n'):
        line = line.strip()
        if len(line) == 0:
            continue
        k, v = line.split('=', 1)
        v = v.strip('"')
        if len(v) > 0:
            env[k] = v
    return env


GOENV = _goenv()
GOPATH = GOENV["GOPATH"]
GOBIN = GOENV["GOPATH"] + '/bin'
GOHOSTOS = GOENV["GOHOSTOS"]
GOHOSTARCH = GOENV["GOHOSTARCH"]
GOC = 'go'


def metadata(cwd, goos='', goarch=''):
    md = {
        'commit_hash': subprocess.check_output('git rev-parse --verify HEAD', shell=True, cwd=cwd).strip(),
        'git_branch': subprocess.check_output('git rev-parse --abbrev-ref HEAD', shell=True, cwd=cwd).strip(),
        # http://stackoverflow.com/a/1404862/3476121
        'git_tag': subprocess.check_output('git describe --exact-match --abbrev=0 2>/dev/null || echo ""', shell=True,
                                           cwd=cwd).strip(),
        'commit_timestamp': datetime.datetime.utcfromtimestamp(
            int(subprocess.check_output('git show -s --format=%ct', shell=True, cwd=cwd).strip())).isoformat(),
        'build_timestamp': datetime.datetime.utcnow().isoformat(),
        'build_host': socket.gethostname(),
        'build_host_os': GOENV["GOHOSTOS"],
        'build_host_arch': GOENV["GOHOSTARCH"]
    }
    if md['git_tag']:
        md['version'] = md['git_tag']
        md['version_strategy'] = 'tag'
    elif not md['git_branch'] in ['master', 'HEAD'] and not md['git_branch'].startswith('release-'):
        md['version'] = md['git_branch']
        md['version_strategy'] = 'branch'
    else:
        hash_ver = subprocess.check_output('git describe --tags --always --dirty', shell=True, cwd=cwd).strip()
        md['version'] = hash_ver
        md['version_strategy'] = 'commit_hash'
    if goos:
        md['os'] = goos
    if goarch:
        md['arch'] = goarch
    return md


def read_file(name):
    with open(name, 'r') as f:
        return f.read()
    return ''


def write_file(name, content):
    dir = os.path.dirname(name)
    if not os.path.exists(dir):
        os.makedirs(dir)
    with open(name, 'w') as f:
        return f.write(content)


def append_file(name, content):
    with open(name, 'a') as f:
        return f.write(content)


def write_checksum(folder, file):
    cmd = "openssl md5 {0} | sed 's/^.* //' > {0}.md5".format(file)
    subprocess.call(cmd, shell=True, cwd=folder)
    cmd = "openssl sha1 {0} | sed 's/^.* //' > {0}.sha1".format(file)
    subprocess.call(cmd, shell=True, cwd=folder)


# TODO: use unicode encoding
def read_json(name):
    try:
        with open(name, 'r') as f:
            return json.load(f, object_pairs_hook=OrderedDict)
    except IOError:
        return {}


def write_json(obj, name):
    with io.open(name, 'w', encoding='utf-8') as f:
        f.write(unicode(json.dumps(obj, indent=2, separators=(',', ': '), ensure_ascii=False)))


def call(cmd, stdin=None, cwd=None):
    print(cmd)
    return subprocess.call([expandvars(cmd)], shell=True, stdin=stdin, cwd=cwd)


def die(status):
    if status:
        sys.exit(status)


def check_output(cmd, stdin=None, cwd=None):
    print(cmd)
    return subprocess.check_output([expandvars(cmd)], shell=True, stdin=stdin, cwd=cwd)


def deps():
    die(call('go get -u golang.org/x/tools/cmd/goimports'))
    die(call('go get -u golang.org/x/tools/cmd/stringer'))
    die(call('go get -u github.com/Masterminds/glide'))
    die(call('go get -u github.com/sgotti/glide-vc'))
    die(call('go get -u github.com/jteeuwen/go-bindata/...'))
    die(call('go get -u github.com/progrium/go-extpoints'))
    die(call('go get -u github.com/tools/godep'))
    die(call('go get -u github.com/uber/go-torch'))


def to_upper_camel(lower_snake):
    components = lower_snake.split('_')
    # We capitalize the first letter of each component
    # with the 'title' method and join them together.
    return ''.join(x.title() for x in components[:])


# ref: https://golang.org/cmd/go/
def go_build(name, goos, goarch, main, compress=False, upx=False):
    linker_opts = []
    if BIN_MATRIX[name].get('go_version', False):
        md = metadata(REPO_ROOT, goos, goarch)
        if md['version_strategy'] == 'tag':
            del md['build_timestamp']
            del md['build_host']
            del md['build_host_os']
            del md['build_host_arch']
        for k, v in md.items():
            linker_opts.append('-X')
            linker_opts.append('main.' + to_upper_camel(k) + '=' + v)

    cgo_env = 'CGO_ENABLED=0'
    cgo = ''
    if BIN_MATRIX[name].get('use_cgo', False):
        cgo_env = "CGO_ENABLED=1"
        cgo = "-a -installsuffix cgo"
        linker_opts.append('-linkmode external -extldflags -static -w')

    ldflags = ''
    if linker_opts:
        ldflags = "-ldflags '{}'".format(' '.join(linker_opts))

    tags = "-tags 'osusergo netgo static_build'"

    bindir = 'dist/{name}'.format(name=name)
    if not os.path.isdir(bindir):
        os.makedirs(bindir)
    if goos == 'alpine':
        repo_dir = REPO_ROOT[len(GOPATH):]
        uid = check_output('id -u').strip()
        cmd = "docker run --rm -u {uid} -v /tmp:/.cache -v {repo_root}:/go{repo_dir} -w /go{repo_dir} -e {cgo_env} golang:1.11-alpine {goc} build -o {bindir}/{name}-{goos}-{goarch}{ext} {cgo} {ldflags} {tags} {main}".format(
            repo_root=REPO_ROOT,
            repo_dir=repo_dir,
            uid=uid,
            name=name,
            goc=GOC,
            goos=goos,
            goarch=goarch,
            bindir=bindir,
            cgo_env=cgo_env,
            cgo=cgo,
            ldflags=ldflags,
            tags=tags,
            ext='.exe' if goos == 'windows' else '',
            main=main
        )
    else:
        cmd = "GOOS={goos} GOARCH={goarch} {cgo_env} {goc} build -o {bindir}/{name}-{goos}-{goarch}{ext} {cgo} {ldflags} {tags} {main}".format(
            name=name,
            goc=GOC,
            goos=goos,
            goarch=goarch,
            bindir=bindir,
            cgo_env=cgo_env,
            cgo=cgo,
            ldflags=ldflags,
            tags=tags,
            ext='.exe' if goos == 'windows' else '',
            main=main
        )
    die(call(cmd, cwd=REPO_ROOT))

    if upx and (goos in ['linux', 'darwin']) and (goarch in ['amd64', '386']):
        cmd = "upx --brute {name}-{goos}-{goarch}{ext}".format(
                name=name,
                goos=goos,
                goarch=goarch,
                bindir=bindir,
                ext='.exe' if goos == 'windows' else ''
            )
        die(call(cmd, cwd=REPO_ROOT + '/' + bindir))

    if compress:
        if goos in ['windows']:
            cmd = "zip {name}-{goos}-{goarch}.zip {name}-{goos}-{goarch}{ext}"
        else:
            cmd = "bzip2 --keep -vf {name}-{goos}-{goarch}{ext}"
        cmd = cmd.format(
                name=name,
                goos=goos,
                goarch=goarch,
                ext='.exe' if goos == 'windows' else ''
            )
        die(call(cmd, cwd=REPO_ROOT + '/' + bindir))
    print('')


def upload_to_cloud(folder, f, version):
    write_checksum(folder, f)
    name = os.path.basename(folder)
    if name not in BIN_MATRIX:
        return
    if ENV == 'prod' and not BIN_MATRIX[name].get('release', False):
        return

    buckets = BUCKET_MATRIX.get(ENV, BUCKET_MATRIX['dev'])
    if not isinstance(buckets, dict):
        buckets = {buckets: ''}
    for bucket, region in buckets.items():
        dst = "{bucket}/binaries/{name}/{version}/{file}".format(
            bucket=bucket,
            name=name,
            version=version,
            file=f
        )
        if bucket.startswith('gs://'):
            upload_to_gcs(folder, f, dst, BIN_MATRIX[name].get('release', False))
        elif bucket.startswith('s3://'):
            upload_to_s3(folder, f, dst, region, BIN_MATRIX[name].get('release', False))


def upload_to_gcs(folder, src, dst, public):
    call("gsutil cp {0} {1}".format(src, dst), cwd=folder)
    call("gsutil cp {0}.md5 {1}.md5".format(src, dst), cwd=folder)
    call("gsutil cp {0}.sha1 {1}.sha1".format(src, dst), cwd=folder)
    if public:
        call("gsutil acl ch -u AllUsers:R {0}".format(dst), cwd=folder)
        call("gsutil acl ch -u AllUsers:R {0}.md5".format(dst), cwd=folder)
        call("gsutil acl ch -u AllUsers:R {0}.sha1".format(dst), cwd=folder)


def upload_to_s3(folder, src, dst, region, public):
    opt_region = ''
    if region:
        opt_region = '--region ' + region
    opt_public = ''
    if public:
        opt_public = "--acl public-read"
    call("aws s3 cp {2} {3} {0} {1}".format(src, dst, opt_region, opt_public), cwd=folder)
    call("aws s3 cp {2} {3} {0}.md5 {1}.md5".format(src, dst, opt_region, opt_public), cwd=folder)
    call("aws s3 cp {2} {3} {0}.sha1 {1}.sha1".format(src, dst, opt_region, opt_public), cwd=folder)


def update_registry(version):
    dist = REPO_ROOT + '/dist'
    bucket = BUCKET_MATRIX.get(ENV, BUCKET_MATRIX['dev'])
    lf = dist + '/latest.txt'
    write_file(lf, version)
    for name in os.listdir(dist):
        if os.path.isfile(dist + '/' + name):
            continue
        if name not in BIN_MATRIX:
            continue
        call("gsutil cp {2} {0}/binaries/{1}/latest.txt".format(bucket, name, lf), cwd=REPO_ROOT)
        if BIN_MATRIX[name].get('release', False):
            call('gsutil acl ch -u AllUsers:R -r {0}/binaries/{1}/latest.txt'.format(bucket, name), cwd=REPO_ROOT)


def ungroup_go_imports(*paths):
    for p in paths:
        if os.path.isfile(p):
            print('Ungrouping imports of file: ' + p)
            _ungroup_go_imports(p)
        elif os.path.isdir(p):
            print('Ungrouping imports of dir: ' + p)
            for dir, _, files in os.walk(p):
                for f in fnmatch.filter(files, '*.go'):
                    _ungroup_go_imports(dir + '/' + f)
        else:
            for f in glob.glob(p):
                print('Ungrouping imports of file: ' + f)
                _ungroup_go_imports(f)


BEGIN_IMPORT_REGEX = ur'import \(\s*'
END_IMPORT_REGEX = ur'\)\s*'


def _ungroup_go_imports(fname):
    with open(fname, 'r+') as f:
        content = f.readlines()
        out = []
        import_block = False
        for line in content:
            c = line.strip()
            if import_block:
                if c == '':
                    continue
                elif re.match(END_IMPORT_REGEX, c) is not None:
                    import_block = False
            elif re.match(BEGIN_IMPORT_REGEX, c) is not None:
                    import_block = True
            out.append(line)
        f.seek(0)
        f.writelines(out)
        f.truncate()

def git_branch_exists(branch):
    return call('git show-ref --quiet refs/heads/{0}'.format(branch), cwd=REPO_ROOT) == 0


def git_checkout(branch):
    call('git fetch --all --prune', cwd=REPO_ROOT)
    call('git fetch --tags', cwd=REPO_ROOT)
    if git_branch_exists(branch):
        call('git checkout {0}'.format(branch), cwd=REPO_ROOT)
    else:
        call('git checkout -b {0}'.format(branch), cwd=REPO_ROOT)


def git_requires_commit():
    changed_files = check_output('git diff --name-only', cwd=REPO_ROOT).strip().split('\n')
    return Counter(changed_files) != Counter(['glide.lock'])


def glide_mod(glide_config):
    for x in REQUIRED_DEPS:
        for idx, dep in enumerate(glide_config['import']):
            found = False
            if dep['package'] == x['package']:
                glide_config['import'][idx] = x
                found = True
                break
        if not found:
            glide_config['import'].append(x)
    for x in DEP_LIST:
        for idx, dep in enumerate(glide_config['import']):
            if dep['package'] == x['package']:
                glide_config['import'][idx] = x
                break


def glide_write(f, glide_config):
    f.seek(0)
    pkg = glide_config.pop('package')
    out = 'package: ' + pkg + '\n' + yaml.dump(glide_config, default_flow_style=False)
    f.write(out)
    f.truncate()
    glide_config['package'] = pkg


REQUIRED_DEPS = [
    {
      "package": "github.com/cpuguy83/go-md2man",
      "version": "v1.0.8"
    },
    {
      "package": "github.com/russross/blackfriday",
      "version": "v1.5.2"
    },
    {
      "package": "github.com/json-iterator/go",
      "version": "1.1.5"
    },
    {
      "package": "github.com/spf13/cobra",
      "version": "v0.0.3"
    },
    {
      "package": "github.com/spf13/pflag",
      "version": "v1.0.3"
    },
    {
      "package": "golang.org/x/text",
      "version": "b19bf474d317b857955b12035d2c5acb57ce8b01"
    },
    {
      "package": "golang.org/x/net",
      "version": "1c05540f6879653db88113bc4a2b70aec4bd491f"
    },
    {
      "package": "golang.org/x/sys",
      "version": "95c6576299259db960f6c5b9b69ea52422860fce"
    },
    {
      "package": "golang.org/x/crypto",
      "version": "de0752318171da717af4ce24d0a2e8626afaeb11"
    },
    {
      "package": "github.com/golang/protobuf",
      "version": "v1.1.0"
    },
    {
      "package": "github.com/davecgh/go-spew",
      "version": "v1.1.1"
    },
    {
      "package": "k8s.io/kube-openapi",
      "version": "0cf8f7e6ed1d2e3d47d02e3b6e559369af24d803"
    },
    {
      "package": "gopkg.in/yaml.v2",
      "version": "v2.2.1"
    },
    {
      "package": "github.com/gorilla/websocket",
      "version": "v1.4.0"
    }
]
DEP_LIST = [
    {
      "package": "github.com/cpuguy83/go-md2man",
      "version": "v1.0.8"
    },
    {
      "package": "github.com/json-iterator/go",
      "version": "1.1.5"
    },
    {
      "package": "github.com/coreos/prometheus-operator",
      "version": "v0.25.0"
    },
    {
      "package": "k8s.io/api",
      "version": "kubernetes-1.12.0"
    },
    {
      "package": "k8s.io/apiextensions-apiserver",
      "version": "kubernetes-1.12.0"
    },
    {
      "package": "k8s.io/apimachinery",
      "repo": "https://github.com/kmodules/apimachinery.git",
      "vcs": "git",
      "version": "ac-1.12.0"
    },
    {
      "package": "k8s.io/apiserver",
      "repo": "https://github.com/kmodules/apiserver.git",
      "vcs": "git",
      "version": "ac-1.12.0"
    },
    {
      "package": "k8s.io/client-go",
      "version": "v9.0.0"
    },
    {
      "package": "k8s.io/cli-runtime",
      "version": "kubernetes-1.12.0"
    },
    {
      "package": "k8s.io/kubernetes",
      "version": "v1.12.0"
    },
    {
      "package": "k8s.io/kube-aggregator",
      "version": "kubernetes-1.12.0"
    },
    {
      "package": "k8s.io/metrics",
      "version": "kubernetes-1.12.0"
    },
    {
      "package": "github.com/appscode/kutil",
      "version": "release-9.0"
    },
    {
      "package": "github.com/appscode/kubernetes-webhook-util",
      "version": "release-9.0"
    },
    {
      "package": "kmodules.xyz/custom-resources",
      "repo": "https://github.com/kmodules/custom-resources.git",
      "vcs": "git",
      "version": "release-9.0"
    },
    {
      "package": "kmodules.xyz/monitoring-agent-api",
      "repo": "https://github.com/kmodules/monitoring-agent-api.git",
      "vcs": "git",
      "version": "release-9.0"
    },
    {
      "package": "kmodules.xyz/objectstore-api",
      "repo": "https://github.com/kmodules/objectstore-api.git",
      "vcs": "git",
      "version": "release-9.0"
    },
    {
      "package": "kmodules.xyz/offshoot-api",
      "repo": "https://github.com/kmodules/offshoot-api.git",
      "vcs": "git",
      "version": "release-9.0"
    },
    {
      "package": "kmodules.xyz/openshift",
      "repo": "https://github.com/kmodules/openshift.git",
      "vcs": "git",
      "version": "release-9.0"
    },
    {
      "package": "github.com/graymeta/stow",
      "repo": "https://github.com/appscode/stow.git",
      "vcs": "git",
      "version": "master"
    },
    {
      "package": "github.com/Azure/azure-sdk-for-go",
      "version": "v19.0.0"
    },
    {
      "package": "github.com/Azure/go-autorest",
      "version": "v10.14.0"
    },
    {
      "package": "github.com/aws/aws-sdk-go",
      "version": "v1.14.12"
    },
    {
      "package": "google.golang.org/api/storage/v1",
      "version": "3639d6d93f377f39a1de765fa4ef37b3c7ca8bd9"
    },
    {
      "package": "cloud.google.com/go",
      "version": "v0.2.0"
    },
    {
      "package": "github.com/spf13/afero",
      "version": "v1.1.2"
    },
    {
      "package": "github.com/appscode/osm",
      "version": "0.9.0"
    },
    {
      "package": "github.com/kubepack/onessl",
      "version": "0.9.0"
    }
]


def revendor():
    seed = ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(6))
    revendor_branch = 'k-1.12.0'
    print(REPO_ROOT)

    call('git reset HEAD --hard', cwd=REPO_ROOT)
    call('git clean -xfd', cwd=REPO_ROOT)
    git_checkout('master')
    call('git pull --rebase origin master', cwd=REPO_ROOT)
    git_checkout(revendor_branch)
    # https://stackoverflow.com/a/6759339/244009
    call("find " + REPO_ROOT + "/apis -type f -exec sed -i -e 's/k8s.io\\/apimachinery\\/pkg\\/api\\/testing\\/roundtrip/k8s.io\\/apimachinery\\/pkg\\/api\\/apitesting\\/roundtrip/g' {} \;")
    with open(REPO_ROOT + '/glide.yaml', 'r+') as glide_file:
        glide_config = yaml.load(glide_file)
        glide_mod(glide_config)
        glide_write(glide_file, glide_config)
        call('glide slow', cwd=REPO_ROOT)
        if git_requires_commit():
            call('git add --all', cwd=REPO_ROOT)
            call('git commit -s -a -m "Update Kubernetes client libraries to 1.12.0"', cwd=REPO_ROOT)
            call('git push origin {0}'.format(revendor_branch), cwd=REPO_ROOT)
        else:
            call('git reset HEAD --hard', cwd=REPO_ROOT)
