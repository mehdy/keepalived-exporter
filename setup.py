from setuptools import find_packages, setup

setup(
    name="keepalived-exporter",
    extras_require=dict(tests=["pytest", "pytest-cov"]),
    packages=find_packages(where="scripts"),
    package_dir={"": "scripts"},
)
