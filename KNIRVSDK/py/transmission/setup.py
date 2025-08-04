from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="knirv-uri-sdk",
    version="0.1.0",
    author="KNIRVCHAIN Team",
    author_email="info@knirvchain.com",
    description="KNIRV Client SDK for Python",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP",
    packages=find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.7",
    install_requires=[
        "py-libp2p>=0.1.5",
        "multibase>=1.0.0",
        "multihash>=0.1.0",
        "base58>=2.1.0",
    ],
)