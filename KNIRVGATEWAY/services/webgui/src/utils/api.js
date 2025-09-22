import axios from 'axios';

   // Default to gateway backend if environment variable is not set
   const API_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';

   console.log('[WebGUI API] Backend URL:', API_URL);

   const api = axios.create({
     baseURL: API_URL,
     headers: {
       'Content-Type': 'application/json',
     },
     timeout: 5000, // 5 second timeout
   });

   export default api;