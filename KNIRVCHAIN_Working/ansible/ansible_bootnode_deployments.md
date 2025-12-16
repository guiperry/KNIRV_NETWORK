```markdown
# Ansible Playbook for KNIRVCHAIN Bootnode Deployment on AWS

This Ansible playbook automates the deployment of a KNIRVCHAIN bootnode on Amazon Web Services (AWS). It handles both the provisioning of the necessary AWS infrastructure and the configuration of the bootnode software.

## Assumptions

*   Ansible is installed and configured with AWS credentials (e.g., via environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, or an IAM role).
*   An existing SSH key pair in AWS is available for accessing the instance.
*   The KNIRVCHAIN executable (`KNIRVCHAIN_executable`) is pre-compiled and available on the Ansible control node.
*   A `config.json` file specifically configured for a bootnode exists (e.g., `IsBootnode: true`, correct `MinersAddress` and `MasterAddress`).
*   The corresponding `wallet.dat` and `master_wallet.dat` files for the bootnode exist.

## Playbook (`deploy_bootnode_aws.yml`)

```yaml
---
- name: Provision KNIRVCHAIN Bootnode Infrastructure on AWS
  hosts: localhost
  gather_facts: false
  vars:
    aws_region: "us-east-1" # Specify your AWS region
    key_pair_name: "your-aws-keypair-name" # Specify your AWS EC2 key pair name
    instance_type: "t3.micro" # Choose an appropriate instance type
    ami_id: "ami-0c7217cdde317cfec" # Example: Ubuntu Server 22.04 LTS (HVM), SSD Volume Type for us-east-1. Find the latest for your region.
                                       # Find the latest for your region.  See below.
    security_group_name: "KNIRVCHAIN-bootnode-sg"
    p2p_port: 6050 # Your KNIRVCHAIN P2P port
    api_port: 6000 # Your KNIRVCHAIN API port
    ssh_port: 22
    instance_tags:
      Name: "KNIRVCHAIN-bootnode"
      Project: "KNIRVCHAIN"
    ansible_ssh_user: "ubuntu" # User for the AMI (e.g., 'ubuntu' for Ubuntu, 'ec2-user' for Amazon Linux)

  tasks:
    - name: Create Security Group for Bootnode
      amazon.aws.ec2_security_group:
        name: "{{ security_group_name }}"
        description: "Security group for KNIRVCHAIN Bootnode"
        region: "{{ aws_region }}"
        rules:
          - proto: tcp
            ports:
              - "{{ ssh_port }}"
            cidr_ip: "0.0.0.0/0" # WARNING: For production, restrict this to your IP
            rule_desc: "Allow SSH"
          - proto: tcp
            ports:
              - "{{ p2p_port }}"
            cidr_ip: "0.0.0.0/0"
            rule_desc: "Allow KNIRVCHAIN P2P TCP"
          - proto: udp # If your P2P also uses UDP (e.g., QUIC)
            ports:
              - "{{ p2p_port }}"
            cidr_ip: "0.0.0.0/0"
            rule_desc: "Allow KNIRVCHAIN P2P UDP"
          - proto: tcp
            ports:
              - "{{ api_port }}"
            cidr_ip: "0.0.0.0/0" # Restrict if API should not be public
            rule_desc: "Allow KNIRVCHAIN API"
        tags: "{{ instance_tags }}"
      register: sg_result

    - name: Launch EC2 Instance for Bootnode
      amazon.aws.ec2_instance:
        key_name: "{{ key_pair_name }}"
        instance_type: "{{ instance_type }}"
        image_id: "{{ ami_id }}"
        region: "{{ aws_region }}"
        security_group: "{{ security_group_name }}"
        tags: "{{ instance_tags }}"
        wait: true
        wait_timeout: 600
        instance_initiated_shutdown_behavior: stop
        volumes:
          - device_name: /dev/sda1
            volume_size: 20 # Adjust disk size as needed
            volume_type: gp3
            delete_on_termination: true
      register: ec2_result

    - name: Allocate and Associate Elastic IP
      when: ec2_result.instances | length > 0
      amazon.aws.ec2_eip:
        region: "{{ aws_region }}"
        device_id: "{{ ec2_result.instances[0].instance_id }}"
        tags: "{{ instance_tags }}"
      register: eip_result

    - name: Add new instance to host group
      when: ec2_result.instances | length > 0
      ansible.builtin.add_host:
        name: "KNIRVCHAIN_bootnode_host"
        ansible_host: "{{ eip_result.public_ip if eip_result.public_ip is defined else ec2_result.instances[0].public_ip_address }}"
        ansible_user: "{{ ansible_ssh_user }}"
        ansible_ssh_private_key_file: "~/.ssh/{{ key_pair_name }}.pem" # Path to your local private key
      changed_when: false

    - name: Wait for SSH to come up
      when: ec2_result.instances | length > 0
      ansible.builtin.wait_for_connection:
        delay: 60
        timeout: 300

    - name: Display Bootnode Public IP
      when: ec2_result.instances | length > 0
      ansible.builtin.debug:
        msg: "Bootnode Public IP: {{ eip_result.public_ip if eip_result.public_ip is defined else ec2_result.instances[0].public_ip_address }}"

- name: Configure KNIRVCHAIN Bootnode Software
  hosts: KNIRVCHAIN_bootnode_host # Target the dynamically added host
  become: true # Most tasks require sudo
  vars:
    KNIRVCHAIN_executable_local_path: "/path/to/your/local/KNIRVCHAIN_executable" # CHANGE THIS
    KNIRVCHAIN_config_local_path: "/path/to/your/local/bootnode_config.json"     # CHANGE THIS
    KNIRVCHAIN_wallet_local_path: "/path/to/your/local/wallet.dat"           # CHANGE THIS
    KNIRVCHAIN_master_wallet_local_path: "/path/to/your/local/master_wallet.dat" # CHANGE THIS

    remote_user: "{{ ansible_ssh_user }}" # e.g., 'ubuntu'
    remote_KNIRVCHAIN_bin_dir: "/opt/KNIRVCHAIN"
    remote_KNIRVCHAIN_app_config_dir: "/home/{{ remote_user }}/.config/KNIRVCHAIN"
    remote_KNIRVCHAIN_data_dir_base: "/home/{{ remote_user }}/.config/KNIRVCHAIN/data" # Base for data, actual path might include ChainID
    service_name: "KNIRVCHAIN-bootnode"

  tasks:
    - name: Update apt cache (for Debian/Ubuntu)
      ansible.builtin.apt:
        update_cache: yes
        cache_valid_time: 3600
      when: ansible_os_family == "Debian"

    - name: Install ca-certificates (dependency for HTTPS calls by the app, if any)
      ansible.builtin.apt:
        name: ca-certificates
        state: present
      when: ansible_os_family == "Debian"

    - name: Create KNIRVCHAIN binary directory
      ansible.builtin.file:
        path: "{{ remote_KNIRVCHAIN_bin_dir }}"
        state: directory
        mode: '0755'

    - name: Create KNIRVCHAIN application config directory
      ansible.builtin.file:
        path: "{{ remote_KNIRVCHAIN_app_config_dir }}"
        state: directory
        owner: "{{ remote_user }}"
        group: "{{ remote_user }}"
        mode: '0750' # Restrict access to config files

    - name: Create KNIRVCHAIN application data base directory
      ansible.builtin.file:
        path: "{{ remote_KNIRVCHAIN_data_dir_base }}"
        state: directory
        owner: "{{ remote_user }}"
        group: "{{ remote_user }}"
        mode: '0750'

    - name: Copy KNIRVCHAIN executable
      ansible.builtin.copy:
        src: "{{ KNIRVCHAIN_executable_local_path }}"
        dest: "{{ remote_KNIRVCHAIN_bin_dir }}/KNIRVCHAIN_executable"
        mode: '0755'

    - name: Copy KNIRVCHAIN config.json for bootnode
      ansible.builtin.copy:
        src: "{{ KNIRVCHAIN_config_local_path }}"
        dest: "{{ remote_KNIRVCHAIN_app_config_dir }}/config.json"
        owner: "{{ remote_user }}"
        group: "{{ remote_user }}"
        mode: '0640'

    - name: Copy KNIRVCHAIN wallet.dat
      ansible.builtin.copy:
        src: "{{ KNIRVCHAIN_wallet_local_path }}"
        dest: "{{ remote_KNIRVCHAIN_app_config_dir }}/wallet.dat"
        owner: "{{ remote_user }}"
        group: "{{ remote_user }}"
        mode: '0600' # Restrict access

    - name: Copy KNIRVCHAIN master_wallet.dat
      ansible.builtin.copy:
        src: "{{ KNIRVCHAIN_master_wallet_local_path }}"
        dest: "{{ remote_KNIRVCHAIN_app_config_dir }}/master_wallet.dat"
        owner: "{{ remote_user }}"
        group: "{{ remote_user }}"
        mode: '0600' # Restrict access

    - name: Create systemd service file for KNIRVCHAIN bootnode
      ansible.builtin.template:
        src: templates/KNIRVCHAIN-bootnode.service.j2
        dest: "/etc/systemd/system/{{ service_name }}.service"
        mode: '0644'
      notify: Reload systemd and restart KNIRVCHAIN-bootnode

  handlers:
    - name: Reload systemd and restart KNIRVCHAIN-bootnode
      ansible.builtin.systemd:
        daemon_reload: true

    - name: Enable and start KNIRVCHAIN-bootnode service
      ansible.builtin.systemd:
        name: "{{ service_name }}"
        enabled: true
        state: restarted # Use restarted to ensure it picks up new config/binary if changed
```

### Finding an AMI ID:

You'll need to find the appropriate AMI ID for your desired AWS region and OS. Here's how to find one for Ubuntu 22.04 LTS in `us-east-1`:

1.  **Go to the AWS Marketplace:**  `https://aws.amazon.com/marketplace`
2.  **Search for "Ubuntu 22.04 LTS":**  Enter that into the search bar.
3.  **Select "Ubuntu Server 22.04 LTS, Free Usage Eligible":**  This is the official Ubuntu AMI.
4.  **Click "Continue to Subscribe":**  Even though it's free, you need to subscribe (no charges apply).
5.  **Click "Continue to Configuration":**
6.  **Choose your Region:** Select your desired AWS region (e.g., "US East (N. Virginia) - us-east-1").
7.  **Find the AMI ID:**  The "AMI ID" will be displayed in this configuration section.  It will look like `ami-0c7217cdde317cfec` (this is just an example; the ID changes over time).

**Important:** AMI IDs are region-specific.  The ID for `us-east-1` will be different than the ID for `eu-west-1`, etc.  Always find the AMI ID *for your target region*.

## Systemd Service Template (`templates/KNIRVCHAIN-bootnode.service.j2`)

Create a directory named `templates` in the same directory as your playbook. Inside it, create the file `KNIRVCHAIN-bootnode.service.j2`:

```jinja
[Unit]
Description=KNIRVCHAIN Bootnode Service
After=network.target

[Service]
User={{ remote_user }}
Group={{ remote_user }}
WorkingDirectory=/home/{{ remote_user }} # Or {{ remote_KNIRVCHAIN_bin_dir }}
ExecStart={{ remote_KNIRVCHAIN_bin_dir }}/KNIRVCHAIN_executable --bootnode
# If your application needs a specific config path when run as a service:
# ExecStart={{ remote_KNIRVCHAIN_bin_dir }}/KNIRVCHAIN_executable --bootnode --config {{ remote_KNIRVCHAIN_app_config_dir }}/config.json

Restart=always
RestartSec=5
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
```

## Before Running:

1.  **Replace Placeholders:**

    *   **In the playbook (`deploy_bootnode_aws.yml`):**
        *   `aws_region`
        *   `key_pair_name`
        *   `ami_id` (ensure it's correct for your chosen `aws_region` and desired OS, e.g., Ubuntu 22.04 LTS).
        *   `ansible_ssh_private_key_file` (path to your local `.pem` file).
        *   `KNIRVCHAIN_executable_local_path`
        *   `KNIRVCHAIN_config_local_path`
        *   `KNIRVCHAIN_wallet_local_path`
        *   `KNIRVCHAIN_master_wallet_local_path`

    *   **In `templates/KNIRVCHAIN-bootnode.service.j2`:**
        *   Review `WorkingDirectory`. The default `WorkingDirectory=/home/{{ remote_user }}` should allow the application to find its config in `~/.config/KNIRVCHAIN/` correctly when run as `{{ remote_user }}`.
        *   If your application doesn't automatically find the config in the user's home when run as a service, uncomment and adjust the `ExecStart` line that includes the `--config` flag.

2.  **AWS Permissions:** Ensure the AWS credentials used by Ansible have permissions to create EC2 instances, security groups, and Elastic IPs.

3.  **File Paths:** Make sure the local paths to your executable and configuration files are correct.

## How to Run:

1.  Save the playbook as `deploy_bootnode_aws.yml` and the template in `templates/KNIRVCHAIN-bootnode.service.j2`.

2.  Run the playbook:

    ```bash
    ansible-playbook deploy_bootnode_aws.yml
    ```

This playbook provides a solid foundation for deploying KNIRVCHAIN bootnodes on AWS. You can extend it further with more sophisticated configuration templating, logging setup, monitoring, etc., as your needs evolve. Remember to test thoroughly in a non-production environment first.  Pay close attention to the AWS region and AMI ID; these are critical for a successful deployment.
```