
# Simplified Ansible Playbook Approach for KNIRVROOT Bootnode Deployment on AWS

This document outlines a simplified Ansible playbook approach for deploying KNIRVROOT bootnodes on AWS, focusing on non-interactive installation and automated deployment triggered by a webhook.

## Playbook Structure

The playbook will consist of two plays:

*   **Play 1: AWS Provisioning:** Remains largely the same, responsible for setting up the EC2 instance, security group, and Elastic IP.
*   **Play 2: Software Installation:** Simplified to install prerequisites, copy the KNIRVROOT executable, and execute it in a non-interactive bootnode installation mode.

## Assumptions

*   Your `KNIRVROOT_executable` has a non-interactive installation mode for setting up a bootnode.  For example: `./KNIRVROOT_executable install --role bootnode --non-interactive` (The exact command might differ based on your application's CLI).
*   This installer, when run, correctly:
    *   Creates `~/.config/KNIRVROOT/` (or the equivalent path for the user the service will run as).
    *   Generates `config.json` with bootnode-specific settings.
    *   Generates `wallet.dat` and `master_wallet.dat` in the configuration directory.
    *   Creates, enables, and starts the systemd service (e.g., `KNIRVROOT-bootnode.service`) configured to run the application as a bootnode (e.g., with the `--bootnode` flag).
    *   Ensures correct file ownership and permissions for configuration files and directories, especially if the installer is run as root but the service runs as a non-root user (e.g., `ubuntu`).

## Proposed Ansible Playbook Changes (Diff)

```yaml
  become: true # Most tasks require sudo
  vars:
    KNIRVROOT_executable_local_path: "/path/to/your/local/KNIRVROOT_executable" # CHANGE THIS

    remote_user: "{{ ansible_user }}" # e.g., 'ubuntu'
    remote_KNIRVROOT_bin_dir: "/opt/KNIRVROOT"
    # The installer within KNIRVROOT_executable is expected to create user-specific config/data dirs
    # e.g., /home/{{ remote_user }}/.config/KNIRVROOT
    service_name: "KNIRVROOT-bootnode"

  tasks:
        state: directory
        mode: '0755'

    - name: Copy KNIRVROOT executable
      ansible.builtin.copy:
        src: "{{ KNIRVROOT_executable_local_path }}"
        dest: "{{ remote_KNIRVROOT_bin_dir }}/KNIRVROOT_executable"
        mode: '0755'

    - name: Run KNIRVROOT installer to set up bootnode
      ansible.builtin.command:
        cmd: "{{ remote_KNIRVROOT_bin_dir }}/KNIRVROOT_executable install --role bootnode --non-interactive"
        # Add other necessary flags for your installer if needed
        # e.g., --user {{ remote_user }} if installer needs to know the target user
      args:
        creates: "/etc/systemd/system/{{ service_name }}.service" # Or another file that indicates successful installation
      changed_when: true # Assume this command always makes changes or reports them
      # Consider adding error handling or checking installer output

    - name: Ensure KNIRVROOT bootnode service is enabled and started
      ansible.builtin.systemd:
        name: "{{ service_name }}" # Ensure this matches the service name created by your installer
        enabled: true
        state: started
      # This task is a safeguard; ideally, your installer handles enabling and starting.
      # If your installer reliably does this, this task might be redundant.

  handlers:
    # Handlers are removed as the installer is expected to manage the service setup and state.
    # If you still need to reload systemd after your installer runs (e.g., if it only creates the file),
+    # you could add a notify to the installer task and a handler for `daemon_reload`.
```

### Explanation of Changes:

*   **Removed Vars:** `KNIRVROOT_config_local_path`, `KNIRVROOT_wallet_local_path`, `KNIRVROOT_master_wallet_local_path` are removed as the installer will generate these.
*   **Removed Directory Creation:** Tasks for creating `remote_KNIRVROOT_app_config_dir` and `remote_KNIRVROOT_data_dir_base` are removed, assuming your installer handles this under the user's home directory (e.g., `/home/ubuntu/.config/KNIRVROOT`).
*   **Removed File Copying:** Tasks for copying `config.json`, `wallet.dat`, and `master_wallet.dat` are removed.
*   **Removed Systemd Template:** The task using `ansible.builtin.template` to create the systemd service file is removed.
*   **Added Installer Command:** A new task `Run KNIRVROOT installer to set up bootnode` is added.
    *   It uses `ansible.builtin.command` to execute your installer.  You'll need to adjust the `cmd` to match the actual command and flags for your non-interactive bootnode installation.
    *   The `args: creates:` line is an example of how to make the task idempotent – it won't run if the specified file (e.g., the systemd service file) already exists.  Adjust this to a reliable indicator of successful installation.
*   **Simplified Service Management:** The handlers are removed. A task `Ensure KNIRVROOT bootnode service is enabled and started` is added as a safeguard. If your installer reliably enables and starts the service, you might even remove this task. The `service_name` variable should match the name your installer uses for the service.
*   **No `templates/KNIRVROOT-bootnode.service.j2` needed:** Since the installer creates the service file, this template is no longer required for the playbook.
*   **Permissions and `become: true`:** When Ansible runs the installer command with `become: true`, the installer will execute as root. Your installer logic needs to:
    *   Create any user-specific directories (like `~/.config/KNIRVROOT`) and ensure they are owned by the `remote_user` (e.g., `ubuntu`).
    *   Ensure the systemd service it creates is configured to run as the `remote_user` (e.g., `User=ubuntu` in the service file).

## Automating Deployment via Webhook (e.g., after payment)

This is a more involved setup but very achievable. Here's a high-level overview:

### Webhook Endpoint:

*   **Purpose:** To receive a notification (HTTP POST request) from your payment processor.
*   **Technology:**
    *   AWS Lambda + API Gateway: A cost-effective, scalable, serverless option.
    *   A small web application (Python Flask/Django, Node.js Express, etc.) running on an EC2 instance or container service.
*   **Logic:**
    *   Validate the webhook request (e.g., using a shared secret, verifying the source IP).
    *   Extract necessary information (e.g., customer ID, plan type if it affects bootnode configuration).

### Triggering Ansible: Once the webhook is validated, the endpoint needs to trigger your Ansible playbook.

*   **Option 1: AWS Systems Manager Run Command:**
    *   The Lambda function (or web app) can use the AWS SDK to invoke an SSM Run Command.
    *   The SSM Document would execute a shell script on a designated "Ansible control" EC2 instance. This script would:
        *   Pull the latest Ansible playbook from a Git repository.
        *   Execute `ansible-playbook deploy_bootnode_aws.yaml`, potentially passing extra variables (`--extra-vars`) derived from the webhook (e.g., `customer_id=...` to tag the instance).

*   **Option 2: CI/CD Pipeline (GitHub Actions, GitLab CI, Jenkins):**
    *   The webhook endpoint triggers a CI/CD pipeline (e.g., via an API call to the CI/CD service).
    *   The pipeline is configured with AWS credentials and Ansible. It checks out the playbook and runs it. This is excellent for version control and auditability.

*   **Option 3: Ansible Tower / AWX API:**
    *   If you use Ansible Tower or AWX, the webhook endpoint can make an API call to launch a predefined Job Template. This is a very robust solution for managing Ansible automation.

*   **Option 4: Direct `ansible-playbook` execution (Simpler, less scalable):**
    *   If the webhook receiver runs on a machine with Ansible and credentials, it could directly execute `subprocess.run(['ansible-playbook', 'deploy_bootnode_aws.yaml', ...])`.

### Parameterization and Idempotency:

*   Your Ansible playbook should be designed to be idempotent (running it multiple times has the same effect as running it once).
*   Pass any dynamic data (like a unique name for the instance based on the customer) as `--extra-vars` to the `ansible-playbook` command.

### Security:

*   Secure your webhook endpoint (HTTPS, authentication/authorization token).
*   Ensure AWS credentials used by Ansible have the least privilege necessary.
*   Store secrets (like API keys for triggering CI/CD) securely (e.g., AWS Secrets Manager, HashiCorp Vault).

### Example Flow (Lambda + SSM Run Command):

```
Payment Processor --(Webhook POST)--> API Gateway --> Lambda Function
                                                          |
                                                          1. Validate Webhook
                                                          2. Extract Data (e.g., customer_id)
                                                          3. Invoke SSM Run Command on "Ansible Control EC2"
                                                             - Command: /usr/local/bin/run_bootnode_playbook.sh {{customer_id}}
                                                                 |
                                                                 V
                                                      Ansible Control EC2
                                                          - run_bootnode_playbook.sh:
                                                            - cd /path/to/ansible/playbooks
                                                            - git pull
                                                            - ansible-playbook deploy_bootnode_aws.yaml --extra-vars "customer_tag=$1"
```

This simplified Ansible playbook, combined with a webhook-triggered mechanism, will give you a powerful automated deployment system for your bootnodes. Remember to thoroughly test your `KNIRVROOT_executable`'s installer to ensure it behaves as expected in a non-interactive, automated context.
```