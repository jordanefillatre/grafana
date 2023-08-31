"""
rgm uses 'github.com/grafana/grafana-build' to build Grafana on the following events:
* A merge to main
* A tag that begins with a 'v'
"""

load(
    "scripts/drone/events/release.star",
    "verify_release_pipeline",
)
load(
    "scripts/drone/pipelines/test_backend.star",
    "test_backend",
)
load(
    "scripts/drone/pipelines/test_frontend.star",
    "test_frontend",
)
load(
    "scripts/drone/pipelines/whats_new_checker.star",
    "whats_new_checker_pipeline",
)
load(
    "scripts/drone/steps/lib.star",
    "get_windows_steps",
)
load(
    "scripts/drone/utils/utils.star",
    "ignore_failure",
    "pipeline",
)
load(
    "scripts/drone/vault.star",
    "from_secret",
    "rgm_dagger_token",
    "rgm_gcp_key_base64",
    "rgm_github_token",
)

docs_paths = {
    "exclude": [
        "*.md",
        "docs/**",
        "packages/**/*.md",
        "latest.json",
    ],
}

tag_trigger = {
    "event": {
        "exclude": [
            "promote",
        ],
    },
    "ref": {
        "include": [
            "refs/tags/v*",
        ],
        "exclude": [
            "refs/tags/*-cloud*",
        ],
    },
}

nightly_trigger = {
    "event": {
        "include": [
            "promote",
            # "cron",
        ],
    },
    "target": {
        "include": [
            "nightly",
        ],
    },
    # "cron": {
    #     "include": [
    #         "nightly-release",
    #     ],
    # },
}

def rgm_build(script = "drone_publish_main.sh", canFail = True, bucket = "grafana-prerelease"):
    rgm_build_step = {
        "name": "rgm-build",
        "image": "grafana/grafana-build:main",
        "commands": [
            "export GRAFANA_DIR=$$(pwd)",
            "cd /src && ./scripts/{}".format(script),
        ],
        "environment": {
            "GCP_KEY_BASE64": from_secret(rgm_gcp_key_base64),
            "GITHUB_TOKEN": from_secret(rgm_github_token),
            "_EXPERIMENTAL_DAGGER_CLOUD_TOKEN": from_secret(rgm_dagger_token),
            "GPG_PRIVATE_KEY": from_secret("packages_gpg_private_key"),
            "GPG_PUBLIC_KEY": from_secret("packages_gpg_public_key"),
            "GPG_PASSPHRASE": from_secret("packages_gpg_passphrase"),
            "DESTINATION": "gs://{}".format(bucket),
        },
        # The docker socket is a requirement for running dagger programs
        # In the future we should find a way to use dagger without mounting the docker socket.
        "volumes": [{"name": "docker", "path": "/var/run/docker.sock"}],
    }
    if canFail:
        rgm_build_step["failure"] = "ignore"

    return [
        rgm_build_step,
    ]

def rgm_main():
    trigger = {
        "event": [
            "push",
        ],
        "branch": "main",
        "paths": docs_paths,
        "repo": [
            "grafana/grafana",
        ],
    }

    return [
        pipeline(
            name = "rgm-main-prerelease",
            trigger = trigger,
            steps = rgm_build(canFail = True),
            depends_on = ["main-test-backend", "main-test-frontend"],
        ),
    ]

def rgm_windows(trigger, name, bucket = "grafana-prerelease"):
    return pipeline(
        name = "rgm-{}-prerelease-windows".format(name),
        trigger = trigger,
        steps = ignore_failure(
            get_windows_steps(
                ver_mode = "release",
                bucket = bucket,
            ),
        ),
        depends_on = ["rgm-{}-prerelease".format(name)],
        platform = "windows",
    )

def rgm_release(trigger, name, bucket = "grafana-prerelease"):
    return [
        test_frontend(trigger, name),
        test_backend(trigger, name),
        pipeline(
            name = "rgm-{}-prerelease".format(name),
            trigger = trigger,
            steps = rgm_build(script = "drone_publish_tag_grafana.sh", canFail = False, bucket = bucket),
            depends_on = ["{}-test-backend".format(name), "{}-test-frontend".format(name)],
        ),
        rgm_windows(trigger, name, bucket),
        verify_release_pipeline(
            trigger = trigger,
            name = "rgm-{}-verify-prerelease-assets".format(name),
            bucket = bucket,
            depends_on = [
                "rgm-{}-prerelease".format(name),
                "rgm-{}-prerelease-windows".format(name),
            ],
        ),
    ]

def rgm():
    return (
        rgm_main() +
        [whats_new_checker_pipeline(tag_trigger)] +
        rgm_release(tag_trigger, "tag", "grafana-prerelease") +
        rgm_release(nightly_trigger, "nightly", "grafana-prerelease/nightly")
    )
