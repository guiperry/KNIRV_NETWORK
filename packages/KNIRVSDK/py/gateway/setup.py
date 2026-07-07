"""
Setup script for the KNIRV Gateway SDK
"""

from setuptools import setup, find_packages

setup(
    name="knirv-gateway-sdk",
    use_scm_version=True,
    setup_requires=["setuptools_scm"],
    package_dir={"": "src"},
    packages=find_packages(where="src"),
    python_requires=">=3.8",
)
