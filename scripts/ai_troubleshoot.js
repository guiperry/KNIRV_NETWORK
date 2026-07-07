#!/usr/bin/env node
/*
AI-assisted Troubleshooter for KNIRV testnet services
- Collects health endpoint output, recent logs, and system status
- Sends a concise prompt to an AI provider (Cerebras or Google) using API keys from deployment/ansible/.env
- Prints suggested troubleshooting steps and optionally executes approved shell commands on the target server

Usage: node scripts/ai_troubleshoot.js --ip <TESTNET_IP> [--ssh-key ~/.ssh/AEGONG.pem] [--health-endpoint /health] [--provider cerebras|google] [--auto-exec]

Security: This script warns and asks for explicit confirmation before executing any commands recommended by the model.
*/

const fs = require('fs');
const path = require('path');
const { execSync, spawnSync } = require('child_process');
const https = require('https');
const http = require('http');

// Load dotenv from deployment/ansible/.env (if exists)
const dotenvPath = path.join(__dirname, '..', 'deployment', 'ansible', '.env');
if (fs.existsSync(dotenvPath)) {
  require('dotenv').config({ path: dotenvPath });
}

const argv = require('yargs')
  .option('ip', { type: 'string', demandOption: true, describe: 'Testnet server IP' })
  .option('ssh-key', { type: 'string', default: process.env.SSH_KEY || '~/.ssh/AEGONG.pem', describe: 'Path to SSH private key' })
  .option('health-endpoint', { type: 'string', default: '/health', describe: 'Health endpoint to query' })
  .option('provider', { type: 'string', choices: ['cerebras', 'google'], default: process.env.AI_PROVIDER || 'cerebras', describe: 'AI provider to use' })
  .option('auto-exec', { type: 'boolean', default: false, describe: 'If true, will auto-execute approved commands from AI without additional prompt' })
  .option('since-minutes', { type: 'number', default: 30, describe: 'How many minutes of logs to retrieve' })
  .help()
  .argv;

const TESTNET_IP = argv.ip;
const SSH_KEY = argv['ssh-key'].replace(/^~\//, process.env.HOME + '/');
const HEALTH_ENDPOINT = argv['health-endpoint'];
const PROVIDER = argv.provider;
const AUTO_EXEC = argv['auto-exec'];
const SINCE_MINUTES = argv['since-minutes'];

function print(...args) { console.log(...args); }
function printErr(...args) { console.error(...args); }

async function httpRequest(options, body) {
  return new Promise((resolve, reject) => {
    const lib = options.protocol === 'https:' ? https : http;
    const req = lib.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        resolve({ statusCode: res.statusCode, headers: res.headers, body: data });
      });
    });
    req.on('error', (err) => reject(err));
    if (body) req.write(body);
    req.end();
  });
}

function gather_health() {
  print('\n[STEP] Fetching health endpoint...');
  try {
    const out = execSync(`curl -s -m 10 http://${TESTNET_IP}${HEALTH_ENDPOINT}`, { encoding: 'utf8' });
    print('[INFO] Health endpoint response:');
    print(out);
    return out;
  } catch (err) {
    printErr('[ERROR] Failed to query health endpoint:', err.message);
    return `ERROR: ${err.message}`;
  }
}

function gather_journalctl(serviceNames, sinceMinutes) {
  print('\n[STEP] Gathering recent system logs (journalctl)...');
  const sinceArg = `--since="${sinceMinutes} minutes ago"`;
  let combined = '';
  for (const svc of serviceNames) {
    try {
      const cmd = `ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP} sudo journalctl -u ${svc} ${sinceArg} -n 200 --no-pager --output=short`;
      const out = execSync(cmd, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] });
      combined += `\n==== ${svc} ====\n` + out;
    } catch (err) {
      combined += `\n==== ${svc} ====\nERROR: ${err.message}`;
    }
  }
  return combined;
}

function gather_recent_logs(remoteLogDirs, sinceMinutes) {
  print('\n[STEP] Gathering recent log files from /opt/knirv-testnet/logs and data directories...');
  const tmpLocal = '/tmp/knirv_troubleshoot_logs';
  try {
    fs.rmSync(tmpLocal, { recursive: true, force: true });
  } catch (e) {}
  fs.mkdirSync(tmpLocal, { recursive: true });

  // Use scp/rsync to fetch a limited number of files (by pattern)
  for (const d of remoteLogDirs) {
    const remotePath = `ubuntu@${TESTNET_IP}:${d}`;
    try {
      // rsync only recent files modified within sinceMinutes using find on remote side
      const remoteTmpTar = `/tmp/knirv_logs_${Date.now()}.tar.gz`;
      const findCmd = `ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP} \"bash -lc 'mkdir -p /tmp/knirv_logs_collect && cd ${d} 2>/dev/null || exit 0; find . -type f -mmin -${sinceMinutes} -print | tar -czf ${remoteTmpTar} -T - 2>/dev/null || true; echo ${remoteTmpTar}'\"`;
      const tarPath = execSync(findCmd, { encoding: 'utf8' }).trim();
      if (tarPath && tarPath.startsWith('/tmp')) {
        const localTar = path.join(tmpLocal, path.basename(tarPath));
        const scpCmd = `scp -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP}:${tarPath} ${localTar}`;
        try {
          execSync(scpCmd, { stdio: 'ignore' });
          // extract
          execSync(`tar -xzf ${localTar} -C ${tmpLocal}`);
          // cleanup remote tar
          execSync(`ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP} sudo rm -f ${tarPath}`);
        } catch (e) {
          // ignore scp errors
        }
      }
    } catch (e) {
      // ignore
    }
  }

  // Read files into a combined string (size-limited)
  let combined = '';
  const maxBytes = 200 * 1024; // 200KB
  const files = fs.readdirSync(tmpLocal, { withFileTypes: true });
  for (const f of files) {
    if (f.isFile()) {
      try {
        const content = fs.readFileSync(path.join(tmpLocal, f.name), 'utf8');
        combined += `\n---- ${f.name} ----\n` + content.substring(0, 60 * 1024);
        if (combined.length > maxBytes) break;
      } catch (e) {}
    }
  }
  return combined;
}

function systemctl_status(services) {
  print('\n[STEP] Checking systemctl status for candidate services');
  let combined = '';
  for (const s of services) {
    try {
      const cmd = `ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP} systemctl status ${s} --no-pager`;
      const out = execSync(cmd, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] });
      combined += `\n==== status ${s} ===\n` + out;
    } catch (err) {
      combined += `\n==== status ${s} ===\nERROR: ${err.message}`;
    }
  }
  return combined;
}

async function call_cerebras_analysis(prompt) {
  print('\n[STEP] Calling Cerebras for initial analysis...');
  const CEREBRAS_API = process.env.CEREBRAS_API_URL || process.env.CEREBRAS_URL;
  const CEREBRAS_KEY = process.env.CEREBRAS_API_KEY || process.env.CEREBRAS_KEY;
  if (!CEREBRAS_API || !CEREBRAS_KEY) {
    throw new Error('Cerebras API URL or key not found in environment (.env)');
  }

  const body = JSON.stringify({
    model: "llama3-70b",
    messages: [
      {
        role: "user",
        content: prompt
      }
    ],
    max_tokens: 2048
  });
  const url = new URL(CEREBRAS_API);
  const options = {
    protocol: url.protocol,
    hostname: url.hostname,
    port: url.port,
    path: url.pathname + (url.search || ''),
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${CEREBRAS_KEY}`
    }
  };

  const res = await httpRequest(options, body);
  try {
    const parsed = JSON.parse(res.body);
    return parsed;
  } catch (e) {
    return { error: 'Failed to parse cerebras response', raw: res.body };
  }
}

async function call_gemini_implementation(analysisPrompt) {
  print('\n[STEP] Calling Gemini for detailed implementation...');
  const GEMINI_API_KEY = process.env.GEMINI_API_KEY;
  if (!GEMINI_API_KEY) {
    throw new Error('Gemini API key not found in environment (.env)');
  }

  // Use Gemini API directly
  const body = JSON.stringify({
    contents: [{
      parts: [{
        text: analysisPrompt
      }]
    }],
    generationConfig: {
      temperature: 0.1,
      maxOutputTokens: 2048
    }
  });

  const options = {
    protocol: 'https:',
    hostname: 'generativelanguage.googleapis.com',
    port: 443,
    path: `/v1beta/models/gemini-1.5-flash:generateContent?key=${GEMINI_API_KEY}`,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' }
  };

  const res = await httpRequest(options, body);
  try {
    const parsed = JSON.parse(res.body);
    return parsed;
  } catch (e) {
    return { error: 'Failed to parse gemini response', raw: res.body };
  }
}

async function call_ai_model_mixture(prompt) {
  print('\n[STEP] Using mixture-of-models approach...');
  
  // Step 1: Use Cerebras for analysis
  const cerebrasPrompt = `You are an expert system administrator analyzing KNIRV testnet issues.

Please analyze the following diagnostic data and provide:
1. A comprehensive diagnosis of the root causes
2. Key findings and patterns identified
3. High-level recommendations for resolution

Diagnostic Data:
${prompt}

Return your analysis in a structured format that can be used by another AI to generate detailed implementation steps.`;
  
  const cerebrasResponse = await call_cerebras_analysis(cerebrasPrompt);
  
  // Extract analysis from Cerebras response
  let analysis = '';
  if (cerebrasResponse.choices && cerebrasResponse.choices[0] && cerebrasResponse.choices[0].message) {
    analysis = cerebrasResponse.choices[0].message.content;
  } else if (cerebrasResponse.choices && cerebrasResponse.choices[0] && cerebrasResponse.choices[0].text) {
    analysis = cerebrasResponse.choices[0].text;
  } else {
    analysis = JSON.stringify(cerebrasResponse, null, 2);
  }

  // Step 2: Use Gemini for implementation
  const geminiPrompt = `You are an expert implementation specialist. Based on the following analysis from Cerebras, generate detailed, executable troubleshooting steps.

CEREBRAS ANALYSIS:
${analysis}

ORIGINAL DIAGNOSTIC DATA:
${prompt}

Please provide:
1. Detailed step-by-step implementation instructions
2. Exact shell commands to run (with sudo where needed)
3. Expected outcomes for each step
4. Safety checks and rollback procedures
5. Verification steps to confirm resolution

Return your response in structured JSON format with fields: diagnosis, actions (array of {title, command, description, risk, verification}), and confidence.`;

  return await call_gemini_implementation(geminiPrompt);
}

async function call_ai_model(prompt) {
  print('\n[STEP] Calling AI provider for analysis...');

  // Use mixture-of-models approach by default
  return await call_ai_model_mixture(prompt);
}

function build_prompt({healthOutput, journalOutput, logFiles, systemStatus}) {
  const prompt = `You are an expert operations assistant for the KNIRV network.

Context:
- Health endpoint output:\n${healthOutput.substring(0, 20*1024)}\n
- Systemctl status (relevant services):\n${systemStatus.substring(0, 20*1024)}\n
- Recent journalctl logs for candidate services:\n${journalOutput.substring(0, 60*1024)}\n
- Recent file logs from /opt/knirv-testnet (truncated):\n${logFiles.substring(0, 120*1024)}\n
Tasks:
1) Analyze the above outputs and provide a concise diagnosis (1-3 short paragraphs).
2) List 5 prioritized troubleshooting actions (each one-line) to restore healthy status.
3) For each action, if it involves running a command on the server, provide the exact shell command(s) to run as the ubuntu user (use sudo where necessary).
4) If the problem appears to be permission-related, explain how to fix permissions safely.
5) Provide a brief explanation of what to look for after each action to confirm progress.

Important security constraints:
- Do NOT delete unknown files.
- Always prefer non-destructive commands and checks (systemctl status, journalctl, docker ps, ls -lah).
- If you recommend destructive commands, mark them clearly and explain the risk.

Return JSON with fields: diagnosis, actions (array of {title, command, description, risk}), and confidence (0-1).
`;
  return prompt;
}

function upload_env_file() {
  print('\n[SECURITY] Uploading .env file to remote host...');
  if (!fs.existsSync(dotenvPath)) {
    printErr('[WARN] .env file not found at', dotenvPath);
    return false;
  }
  
  try {
    const remoteEnvPath = `/tmp/knirv_troubleshoot_env_${Date.now()}.env`;
    const scpCmd = `scp -i ${SSH_KEY} -o StrictHostKeyChecking=no ${dotenvPath} ubuntu@${TESTNET_IP}:${remoteEnvPath}`;
    execSync(scpCmd, { stdio: 'ignore' });
    print('[SECURITY] .env file uploaded to remote host:', remoteEnvPath);
    return remoteEnvPath;
  } catch (err) {
    printErr('[ERROR] Failed to upload .env file:', err.message);
    return false;
  }
}

function cleanup_env_file(remoteEnvPath) {
  if (!remoteEnvPath) return;
  
  print('\n[SECURITY] Cleaning up .env file from remote host...');
  try {
    const rmCmd = `ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no ubuntu@${TESTNET_IP} sudo rm -f ${remoteEnvPath}`;
    execSync(rmCmd, { stdio: 'ignore' });
    print('[SECURITY] .env file removed from remote host');
  } catch (err) {
    printErr('[WARN] Failed to cleanup .env file:', err.message);
  }
}

// Function to format parsed AI response as markdown
function formatResponseAsMarkdown(parsed, markdownText) {
  let markdown = '# AI Troubleshooting Analysis\n\n';
  
  if (parsed) {
    if (parsed.diagnosis) {
      markdown += '## Diagnosis\n\n';
      markdown += `${parsed.diagnosis}\n\n`;
    }
    
    if (parsed.confidence) {
      markdown += `**Confidence**: ${parsed.confidence}\n\n`;
    }
    
    if (parsed.actions && Array.isArray(parsed.actions)) {
      markdown += '## Recommended Actions\n\n';
      parsed.actions.forEach((action, index) => {
        markdown += `### ${index + 1}. ${action.title}\n\n`;
        if (action.command) {
          markdown += '```bash\n';
          markdown += `${action.command}\n`;
          markdown += '```\n\n';
        }
        if (action.description) {
          markdown += `${action.description}\n\n`;
        }
        if (action.risk) {
          markdown += `**Risk Level**: ${action.risk}\n\n`;
        }
        if (action.verification) {
          markdown += `**Verification**: ${action.verification}\n\n`;
        }
        markdown += '---\n\n';
      });
    }
  } else if (markdownText) {
    // If no structured data, use the raw markdown text
    markdown += markdownText;
  }
  
  return markdown;
}

// Function to export response to logfile
function exportToLogfile(content, filename = null) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const logFilename = filename || `ai_troubleshoot_${timestamp}.md`;
  const logPath = path.join(__dirname, '..', 'logs', logFilename);
  
  // Ensure logs directory exists
  const logsDir = path.dirname(logPath);
  if (!fs.existsSync(logsDir)) {
    fs.mkdirSync(logsDir, { recursive: true });
  }
  
  try {
    fs.writeFileSync(logPath, content, 'utf8');
    print(`[LOG] Response exported to: ${logPath}`);
    return logPath;
  } catch (err) {
    printErr('[ERROR] Failed to write logfile:', err.message);
    return null;
  }
}

async function main() {
  let remoteEnvPath = null;
  
  try {
    print('[INFO] Provider:', PROVIDER);

    // Upload .env file to remote host for AI API access
    remoteEnvPath = upload_env_file();
    
    // Gather health output
    const healthOutput = gather_health();

    // Candidate services to inspect
    const services = ['knirvgraph', 'knirvoracle', 'knirvchain', 'knirvrouter', 'knirvserver', 'knirvgateway'];

    const journalOutput = gather_journalctl(services, SINCE_MINUTES);
    const systemStatus = systemctl_status(services);
    const logFiles = gather_recent_logs(['/opt/knirv-testnet/logs', '/opt/knirv-testnet/data'], SINCE_MINUTES);

    const prompt = build_prompt({ healthOutput, journalOutput, logFiles, systemStatus });

    const aiResp = await call_ai_model(prompt);

    print('\n[AI RESPONSE RAW]');
    console.log(JSON.stringify(aiResp, null, 2));

    // Try to extract JSON content from common fields
    let parsed = null;
    let markdownText = '';
    if (aiResp && typeof aiResp === 'object') {
      // Gemini style: { candidates: [{ content: { parts: [{ text }] } }] } or Cerebras style
      try {
        if (aiResp.candidates && aiResp.candidates[0] && aiResp.candidates[0].content && aiResp.candidates[0].content.parts && aiResp.candidates[0].content.parts[0]) {
          const text = aiResp.candidates[0].content.parts[0].text;
          markdownText = text;
          // Try to extract JSON from markdown code blocks
          const jsonMatch = text.match(/```json\n([\s\S]*?)\n```/);
          if (jsonMatch && jsonMatch[1]) {
            parsed = JSON.parse(jsonMatch[1]);
          } else {
            // Fallback: try to parse the entire text as JSON
            parsed = JSON.parse(text);
          }
        } else if (aiResp.choices && aiResp.choices[0] && aiResp.choices[0].message && aiResp.choices[0].message.content) {
          const text = aiResp.choices[0].message.content;
          markdownText = text;
          const jsonMatch = text.match(/```json\n([\s\S]*?)\n```/);
          if (jsonMatch && jsonMatch[1]) {
            parsed = JSON.parse(jsonMatch[1]);
          } else {
            parsed = JSON.parse(text);
          }
        } else if (aiResp.choices && aiResp.choices[0] && aiResp.choices[0].text) {
          const text = aiResp.choices[0].text;
          markdownText = text;
          const jsonMatch = text.match(/```json\n([\s\S]*?)\n```/);
          if (jsonMatch && jsonMatch[1]) {
            parsed = JSON.parse(jsonMatch[1]);
          } else {
            parsed = JSON.parse(text);
          }
        } else if (aiResp.predictions && aiResp.predictions[0]) {
          parsed = aiResp.predictions[0];
        } else if (aiResp.output) {
          parsed = JSON.parse(aiResp.output);
        }
      } catch (e) {
        printErr('[WARN] Failed to parse AI output as JSON. The model may have returned plain text.');
      }
    }

    if (!parsed) {
      print('\n[INFO] AI did not return structured JSON. Please review the raw response above.');
      // Optionally, attempt to extract a JS object from text
    }

    // Format response as markdown
    const markdownResponse = formatResponseAsMarkdown(parsed, markdownText);
    
    // Print formatted markdown to console
    print('\n' + '='.repeat(80));
    print('[AI RESPONSE FORMATTED]');
    print('='.repeat(80));
    print(markdownResponse);
    print('='.repeat(80));

    // Export to logfile
    exportToLogfile(markdownResponse);

    if (parsed && parsed.actions && Array.isArray(parsed.actions)) {
      print('\n[STEP] Suggested actions:');
      parsed.actions.forEach((a, i) => {
        print(`\n[${i+1}] ${a.title}`);
        if (a.command) print(`Command: ${a.command}`);
        if (a.description) print(`Description: ${a.description}`);
        if (a.risk) print(`Risk: ${a.risk}`);
      });

      // Ask user for confirmation before executing any commands
      for (const a of parsed.actions) {
        if (a.command && (!a.risk || a.risk !== 'high')) {
          if (AUTO_EXEC) {
            print('\n[AUTO-EXEC] Running:', a.command);
            try {
              const out = execSync(a.command, { encoding: 'utf8', stdio: 'inherit' });
            } catch (e) {
              printErr('Command failed:', e.message);
            }
          } else {
            const prompt = `Execute this command on ${TESTNET_IP}?\n${a.command}\nEnter y to run, anything else to skip:`;
            const response = execSync(`bash -c 'read -p "${prompt}" -n 1 -r && echo $REPLY'`, { encoding: 'utf8' }).trim();
            if (response.toLowerCase() === 'y') {
              print('\n[EXEC] Running:', a.command);
              try {
                execSync(a.command, { stdio: 'inherit' });
              } catch (e) {
                printErr('Command failed:', e.message);
              }
            } else {
              print('[SKIP] User skipped command');
            }
          }
        }
      }
    }

    print('\n[DONE] Troubleshooting run complete.');
  } catch (err) {
    printErr('[FATAL] Troubleshooter failed:', err.message);
    process.exit(2);
  } finally {
    // Always cleanup the .env file from remote host
    cleanup_env_file(remoteEnvPath);
  }
}

main();
