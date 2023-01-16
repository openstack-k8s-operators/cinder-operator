#!/usr/bin/env python3
# Script to get dependencies recursively for a PR.
# PR message and commit messages must use Depends-On in one of these 4 formats:
# - Depends-On: lib-common=88
# - Depends-On: openstack-k8s-operators/lib-common#88
# - Depends-On: lib-common#88
# - Depends-On: https://github.com/openstack-k8s-operators/lib-common/88
# Output is consumable directly by the setdeps.py script
# $ ./showdeps PR# [repo]
# eg:
#  $ ./showdeps 65
#  lib-common=88
#
#  $ ./showdeps 65 cinder-operator
#  lib-common=88

import json
import re
import sys
import urllib.request

GH_API_URL = 'https://api.github.com/repos/openstack-k8s-operators'
REPO = 'cinder-operator'
RECURSIVE = False


def get_gh_json(repo, pr, ending=''):
    api_url = f'{GH_API_URL}/{repo}/pulls/{pr}{ending}'
    contents_str = urllib.request.urlopen(api_url).read()
    return json.loads(contents_str)


def find_dependencies(text):
    result = []
    depends = re.findall(r"\n\s*Depends-On:\s*(\S+)\s*?", text, re.IGNORECASE)
    for dep in depends:
        # lib-common=88
        if '=' in dep:
            res = dep.split('=')
        # openstack-k8s-operators/lib-common#88
        # lib-common#88
        elif '#' in dep:
            res = dep.rsplit('#', 1)
            res[0] = res[0].rsplit('/', 1)[-1]
        # https://github.com/openstack-k8s-operators/lib-common/88
        else:
            r = dep.rsplit('/', 3)
            if len(r) < 4:
                sys.stderr.write(f'Wrong Depends-On on: {dep}\n')
                continue
            res = r[1], r[3]
        result.append(res)
    return result


def get_dependencies(repo, pr):
    contents = get_gh_json(repo, pr)
    pr_message = contents['body']
    # Initialize to the dependencies found in the PR message
    result = find_dependencies(pr_message)

    # Find additional dependencies in commit messages
    contents = get_gh_json(repo, pr, '/commits')
    for commit in contents:
        message = commit['commit']['message']
        deps = find_dependencies(message)
        if deps:
            result.extend(deps)
    return result


def main(args):
    pr = args[1]
    repo = args[2] if len(args) > 2 else REPO
    dependencies = []
    to_check = [(repo, pr)]
    checked = []
    while to_check:
        check = to_check.pop()
        # Detect circular references
        if check in checked:
            continue
        new_deps = get_dependencies(*check)
        dependencies.extend(new_deps)
        if not RECURSIVE:
            break
        to_check.extend(new_deps)
        checked.append(check)

    if dependencies:
        # Convert to str and remove duplicated dependencies
        deps_str = set(f'{dep[0]}={dep[1]}' for dep in dependencies)
        print(' '.join(deps_str))


if __name__ == '__main__':
    main(sys.argv)
