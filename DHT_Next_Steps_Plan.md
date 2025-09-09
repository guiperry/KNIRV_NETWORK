# DHT Next Steps Plan

This document outlines the manual steps required to complete the Private DHT Deployment Plan implementation. The codebase has been fully implemented and is ready for deployment.

## ✅ Completed Implementation

The following components have been fully implemented:

1. **Private DHT with libp2p** - Complete libp2p-based DHT manager
2. **Unified Server Architecture** - Single codebase supporting multiple deployment modes
3. **Provision Endpoint** - Dynamic peer discovery for all deployment modes
4. **Frontend Failover Logic** - Health check and automatic redirection
5. **Content Synchronization** - Makefile-based sync system
6. **CloudFlare DNS Management** - Automated DNS failover and leader election
7. **Deployment Configurations** - Ready-to-use configs for all platforms

## 🚀 Phase 1: Initial Setup and Dependencies

### 1.1 Install Dependencies
```bash
cd KNIRVGATEWAY
npm install
```

### 1.2 Test Local Implementation
```bash
# Test the unified server in persistent mode
npm run start:render

# In another terminal, test the provision endpoint
curl http://localhost:8080/provision
curl http://localhost:8080/health
```

## 🌐 Phase 2: Domain and DNS Setup

### 2.1 CloudFlare Configuration

1. **Register knirv.network domain** (if not already done)
2. **Add domain to CloudFlare**
3. **Get CloudFlare credentials:**
   - Zone ID: Found in CloudFlare dashboard → Domain → Overview
      CLOUDFLARE_ZONE_ID: 6ec647f95e75c97a504c40a5e07e4e52
   - API Token: CloudFlare dashboard → My Profile → API Tokens → Create Token
   CLOUDFLARE_API_TOKEN: 7d4Wpds92sjiRCwg8bKRmTGLuYtdWhcYyBU88ZMa
   - Use "Custom token" with Zone:Edit permissions for knirv.network

### 2.2 Create DNS Records

Create these initial DNS records in CloudFlare:

```
Type: A
Name: gateway.knirv.network
Content: [Your Render instance IP - will be updated automatically]
TTL: 60 seconds
```

## 🖥️ Phase 3: Server Deployments

### 3.1 Deploy to Render (Persistent Gateway)

1. **Create Render account** and connect GitHub repository
2. **Create new Web Service:**
   - Repository: Your KNIRVGATEWAY repository
   - Branch: main
   - Build Command: `npm install && npm run build`
   - Start Command: `npm run start:render`
   - Environment: Node.js

3. **Set Environment Variables in Render:**
   ```
   NODE_ENV=production
   GATEWAY_MODE=persistent
   KNIRV_CHAIN_ID=testnet
   PORT=8080
   DHT_PORT=4001
   CLOUDFLARE_API_TOKEN=7d4Wpds92sjiRCwg8bKRmTGLuYtdWhcYyBU88ZMa
   CLOUDFLARE_ZONE_ID=6ec647f95e75c97a504c40a5e07e4e52
   INTERNAL_API_KEY=Keperu100
   INSTANCE_IP=[Will be auto-detected or set manually]
   KNIRV_BOOTSTRAP_PEERS=[Comma-separated list of bootstrap peers]
   ```

4. **Deploy and note the Render URL** (e.g., `https://knirvgateway-persistent.onrender.com`)

### 3.2 Deploy to Netlify (Serverless Gateway)

1. **Create Netlify account** and connect GitHub repository
2. **Create new site from Git:**
   - Repository: Your KNIRVGATEWAY repository
   - Branch: main
   - Build command: `npm run build`
   - Publish directory: `.`

3. **Set Environment Variables in Netlify:**
   ```
   NODE_ENV=production
   GATEWAY_MODE=serverless
   KNIRV_CHAIN_ID=testnet
   RENDER_GATEWAY_INTERNAL_API=[Render URL]/internal/peers
   INTERNAL_API_KEY=[Same key as Render]
   ```

4. **Deploy and note the Netlify URL**

### 3.3 Deploy to Vercel (Serverless Gateway)

1. **Create Vercel account** and connect GitHub repository
2. **Import project:**
   - Repository: Your KNIRVGATEWAY repository
   - Framework: Other
   - Build Command: `npm run build`
   - Output Directory: `.`

3. **Set Environment Variables in Vercel:**
   ```
   NODE_ENV=production
   GATEWAY_MODE=serverless
   KNIRV_CHAIN_ID=testnet
   RENDER_GATEWAY_INTERNAL_API=[Render URL]/internal/peers
   RENDER_GATEWAY_URL=[Render URL]
   INTERNAL_API_KEY=[Same key as Render]
   ```

4. **Deploy and note the Vercel URL**

## 🌍 Phase 4: A2 Hosting Setup (knirv.com)

### 4.1 Create knirvcom-repo Repository

1. **Create new GitHub repository** named `knirvcom-repo`
2. **Add the public-facing website content:**
   - Copy `KNIRVGATEWAY/home.html` to `knirvcom-repo/index.html`
   - Copy assets and images directories
   - Create a simple Node.js server (see implementation in PrivateDHTDeploymentPlan.md)

### 4.2 Setup A2 Hosting

1. **Configure Node.js application** on A2 Hosting
2. **Clone knirvcom-repo** to the hosting environment
3. **Set up GitHub webhook:**
   - Repository Settings → Webhooks → Add webhook
   - Payload URL: `https://knirv.com/webhook/github`
   - Content type: `application/json`
   - Secret: [Generate secure secret]
   - Events: Just the push event

4. **Set environment variables:**
   ```
   GITHUB_WEBHOOK_SECRET=[Your webhook secret]
   PORT=3000
   ```

### 4.3 Add knirvcom-repo as Submodule

```bash
cd KNIRVGATEWAY
git submodule add https://github.com/KNIRV-NETWORK/knirvcom-repo.git knirvcom-repo
git submodule update --init --recursive
make sync-failover-page
```

## 🔧 Phase 5: Configuration and Testing

### 5.1 Update DNS Configuration

1. **Update CloudFlare DNS record** to point to Render instance
2. **Set up health monitoring** in CloudFlare (optional)
3. **Configure low TTL** (60 seconds) for fast failover

### 5.2 Test All Endpoints

```bash
# Test Render (persistent)
curl https://[render-url]/provision
curl https://[render-url]/health
curl https://[render-url]/dht/status

# Test Netlify (serverless)
curl https://[netlify-url]/provision
curl https://[netlify-url]/.netlify/functions/provision

# Test Vercel (serverless)
curl https://[vercel-url]/provision
curl https://[vercel-url]/api/provision

# Test knirv.com
curl https://knirv.com/
```

### 5.3 Test Failover Logic

1. **Access any gateway URL** - should redirect to knirv.com
2. **Simulate knirv.com downtime** - should fallback to local home.html
3. **Test DNS failover** by stopping Render instance

## 📊 Phase 6: Monitoring and Maintenance

### 6.1 Set Up Monitoring

1. **CloudFlare Analytics** - Monitor DNS queries and performance
2. **Render Metrics** - Monitor persistent gateway health
3. **Netlify/Vercel Logs** - Monitor serverless function performance

### 6.2 Regular Maintenance Tasks

```bash
# Update dependencies
npm audit fix

# Sync failover content
make sync-failover-page

# Test provision endpoints
make test-provision

# Check system status
make status
```

## 🔐 Security Considerations

### 6.1 API Key Management

- Store all API keys securely in platform environment variables
- Rotate CloudFlare API tokens regularly
- Use different INTERNAL_API_KEY for each environment if needed

### 6.2 Network Security

- Configure firewall rules for DHT ports (4001)
- Use HTTPS for all communications
- Implement rate limiting on provision endpoints

## 🚨 Troubleshooting

### Common Issues

1. **Provision endpoint returns empty array:**
   - Check RENDER_GATEWAY_INTERNAL_API configuration
   - Verify INTERNAL_API_KEY matches across deployments
   - Check Render instance health

2. **DNS failover not working:**
   - Verify CloudFlare API credentials
   - Check INSTANCE_IP configuration
   - Review health check logs

3. **Frontend not redirecting:**
   - Check knirv.com availability
   - Verify CORS configuration
   - Test JavaScript console for errors

### Debug Commands

```bash
# Check DHT status
curl https://[render-url]/dht/status

# Test internal API
curl -H "Authorization: Bearer [INTERNAL_API_KEY]" https://[render-url]/internal/peers

# Check CloudFlare DNS
dig gateway.knirv.network
```

## 📈 Next Phase Enhancements

After successful deployment, consider:

1. **Load balancing** across multiple Render instances
2. **Geographic distribution** of gateways
3. **Advanced monitoring** with custom dashboards
4. **Automated testing** of failover scenarios
5. **Performance optimization** based on metrics

## 📞 Support

If you encounter issues during deployment:

1. Check the implementation logs in each platform
2. Verify all environment variables are set correctly
3. Test each component individually before integration
4. Review the original PrivateDHTDeploymentPlan.md for detailed specifications

The implementation is complete and ready for deployment following these steps!
