#!/usr/bin/env python3
# Script to replace go module dependencies for Cinder:
# Supports 4 different formats:
# - By PR numbers:
#      ./setdeps.py lib-common=88 openstack-operator=38
# - By local directory where the local dir is already with the right code:
#      ./setdeps.py lib-common=../lib-common \
#        openstack-operator=../openstack-operator
# - By upstream repository
#      ./setdeps.py \
#        lib-common=https://github.com/fmount/lib-common/@extra_volumes \
#        openstack-operator=https://github.com/fmount/openstack-operator/@extra_volumes
# - By short upstream repository:
#      ./setdeps.py lib-common=fmount/lib-common@extra_volumes \
#        openstack-operator=fmount/openstack-operator@extra_volumes
import json
import os
import re
import subprocess
import sys
import urllib.request

GH_API_URL = 'https://api.github.com/repos/openstack-k8s-operators'


def get_modules(repo, requires):
    """Return a list of tuples of (repository, module) for a specific repo."""
    result = []
    for require in requires:
        try:
            index = require.index('/' + repo)

            module_index = index + 1 + len(repo)
            url = require[:module_index]
            module = require[module_index:]
            if module[0] == '/':
                result.append((url, module))
        except ValueError:
            continue
    return result


def get_requires():
    """Get all requires modules from go.mod"""
    # Get location of the go.mod file
    result = subprocess.run('go env GOMOD', shell=True, check=True,
                            capture_output=True)
    go_mod_path = result.stdout.strip()
    # Use findall to get direct and indirect
    require_sections = re.findall(r"^require \(\n(.*?)\n\)",
                                  open(go_mod_path).read(),
                                  re.MULTILINE | re.DOTALL)
    if not require_sections:
        raise Exception('Error parsing go.mod')
    res = []
    for requires_section in require_sections:
        lines = requires_section.split('\n')
        res.extend(line.strip().split()[0] for line in lines if line)
    return res


def main(args):
    source_is = None

    requires = get_requires()
    for arg in args:
        repo = arg[0]
        go_module_url = f'github.com/openstack-k8s-operators/{repo}'

        source_version = ''

        try:
            pr = int(arg[1])
            source_is = 'PR'
        except ValueError:
            src_path = arg[1]
            if os.path.exists(src_path):
                source_is = 'LOCAL'
            else:
                source_is = 'REPO'
                # Build url if we where just provided a partial repo
                # eg: fpantano/lib-common@extravol
                if not src_path.startswith('http'):
                    src_path = 'github.com/' + src_path

        if source_is == 'PR':
            api_url = f'{GH_API_URL}/{repo}/pulls/{pr}'

            contents_str = urllib.request.urlopen(api_url).read()
            contents = json.loads(contents_str)
            source_version = '@' + contents['head']['ref']

            src_path = 'github.com/' + contents['head']['repo']['full_name']
            print(f'Source for {repo} PR#{pr} is {src_path}{source_version}')

        elif source_is == 'LOCAL':
            src_path = os.path.abspath(src_path)
            print(f'Source repo for {repo} is {src_path}')
        elif source_is == 'REPO':  # is a repo
            if '@' in src_path:
                src_path, source_version = src_path.split('@')
                source_version = '@' + source_version
            print(f'Source repo for {repo} is {src_path}')

        if source_version:
            print(f'Checking the go mod version for branch {source_version}')
            sha = contents['head']['sha'][:12]
            result = subprocess.run(f'go list -m -json {src_path}@{sha}',
                                    shell=True, check=True,
                                    capture_output=True)
            source_version = '@' + json.loads(result.stdout)['Version']

        modules = get_modules(repo, requires)
        for go_module_url, module in modules:
            cmd = (f'go work edit -replace {go_module_url}{module}='
                   f'{src_path}{module}{source_version}')
            print(cmd)
            os.system(cmd)
        print()


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print('Error, missing arguments')
        exit(1)

    args = []
    for arg in sys.argv[1:]:
        args.append(arg.split('='))

    main(args)
