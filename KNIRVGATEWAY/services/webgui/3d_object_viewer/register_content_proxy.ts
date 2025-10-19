import express, { Request, Response, NextFunction } from 'express';
import axios, { AxiosError } from 'axios';
import * as dotenv from 'dotenv'; // Load environment variables
import { URL } from 'url';

// Load environment variables from .env file (if it exists)
dotenv.config();

interface Config {
    port: string;
    verifierNodes: string[];
    healthCheckInterval: number; //In milliseconds
}

let config: Config = {
    port: process.env.PORT || '3000',
    verifierNodes: process.env.VERIFIER_NODES?.split(',') || [],
    healthCheckInterval: parseInt(process.env.HEALTH_CHECK_INTERVAL || '30000', 10), //Default to 30 seconds if not set.
};

interface VerifierHealth {
    uri: string;
    healthy: boolean;
    latency: number; // in milliseconds
    lastCheck: number; // Timestamp
}

const verifierStatus: { [uri: string]: VerifierHealth } = {};

// Load Config and Ensure ENV Variables are set.
const loadConfig = () => {
    if (!process.env.VERIFIER_NODES) {
        console.error("FATAL ERROR: VERIFIER_NODES environment variable is required.");
        (process.exit as (code?: number) => never)(1);
    }

    try{
        config = {
            port: process.env.PORT || '3000',
            verifierNodes: process.env.VERIFIER_NODES?.split(',') || [],
            healthCheckInterval: parseInt(process.env.HEALTH_CHECK_INTERVAL || '30000', 10)
        };
    } catch (e){
        console.error("Unable to Properly configure ENV Variables. Check the formatting.");
        (process.exit as (code?: number) => never)(1);
    }

    console.log("Configuration Loaded: ", config);

}

//Function to check URL to determine validity.
const isValidURL = (testString: string) => {
    try {
      new URL(testString);
      return true;
    } catch (e) {
      return false;
    }
};

// Function to check the health of a verifier node.
const checkVerifierHealth = async (uri: string): Promise<void> => {
    const startTime = Date.now();

    try {
        if (!isValidURL(uri)) {
            console.error("Invalid URL:", uri);
            verifierStatus[uri] = { uri: uri, healthy: false, latency: 0, lastCheck: Date.now() };
            return;
        }

        const response = await axios.get(uri + '/health', { timeout: 5000 }); // Add timeout, test fast
        const latency = Date.now() - startTime;

        const healthy = response.status >= 200 && response.status < 300;
        verifierStatus[uri] = { uri: uri, healthy: healthy, latency: latency, lastCheck: Date.now() };

    } catch (error: unknown) {
        const latency = Date.now() - startTime;
        console.error(`Error checking health for ${uri}:`, error instanceof Error ? error.message : String(error));

        if (error instanceof AxiosError && error.code === 'ECONNABORTED') {
            console.log("Request timed out to:", uri);
        }

        verifierStatus[uri] = { uri: uri, healthy: false, latency: latency, lastCheck: Date.now() };
    }
};

// Function to update verifier health periodically
const updateVerifierHealthPeriodically = () => {
    config.verifierNodes.forEach(uri => {
        if(!isValidURL(uri)) {
            console.error("Invalid URL:", uri);
            return;
        }

        setInterval(() => {
            checkVerifierHealth(uri);
        }, config.healthCheckInterval);
    });
};

// Function to get the best (healthiest and lowest latency) verifier
const getBestVerifier = (): string | undefined => {
    const healthyVerifiers = Object.values(verifierStatus).filter(status => status.healthy);

    if (healthyVerifiers.length === 0) {
        console.warn("No healthy verifiers available.");
        if (config.verifierNodes.length > 0){
            const randomIndex = Math.floor(Math.random() * config.verifierNodes.length);
            console.warn("Fallback case: Random URI Selected", config.verifierNodes[randomIndex]);
            return config.verifierNodes[randomIndex];
        }
        return undefined; // Handle fallback to undefined;
    }

    healthyVerifiers.sort((a, b) => a.latency - b.latency);
    return healthyVerifiers[0].uri; //Return the first URI after having already check that there are valid cases.
};

const app = express();
app.use(express.json());

// registerURIHandler function to handle /register-uri endpoint
const registerURIHandler = async (req: Request, res: Response) => {
    const bestVerifier = getBestVerifier();
    if (!bestVerifier) {
        return res.status(503).send({ error: 'No available verifiers.' });
    }

    try {
        const proxyResponse = await axios.post(bestVerifier + '/register-uri', req.body, {
            headers: { 'Content-Type': 'application/json' },
        });

        res.status(proxyResponse.status).json(proxyResponse.data);

    } catch (error: unknown) {
        if (error instanceof AxiosError) {
            // Axios-specific error handling for better logging
            console.error(`Error proxying to ${bestVerifier}:`, error.message, error.response?.status, error.response?.data);
            return res.status(error.response?.status || 500).json({ error: `Proxy request failed: ${error.message}` });
        } else {
            // Generic error handling
            console.error("General Error: ", error);
            return res.status(500).json({ error: 'Internal server error' });
        }
    }
};

// Register the /register-uri route
app.post('/register-uri', (req: Request, res: Response) => {
    registerURIHandler(req, res).catch(err => {
        console.error('Unhandled error in registerURIHandler:', err);
        res.status(500).json({ error: 'Internal server error' });
    });
});

//Basic Startup verification to ensure that things are running as designed.
app.get('/status', (req: Request, res: Response) => {
    res.status(200).send("Proxy is Online!");
});

loadConfig();
updateVerifierHealthPeriodically();

app.listen(config.port, () => {
    console.log(`Proxy listening on port ${config.port}`);
});