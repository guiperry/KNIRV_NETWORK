import axios from 'axios';

   export default async function handler(req, res) {
     if (req.method !== 'GET') {
       return res.status(405).json({ message: 'Method not allowed' });
     }

     try {
       const response = await axios.get(`${process.env.NEXT_PUBLIC_BACKEND_URL}/health`);
       return res.status(200).json({
         status: response.data.status,
         isRunning: response.data.status === 'ok'
       });
     } catch (error) {
       console.error('Error checking backend status:', error);
       return res.status(200).json({ 
         status: 'offline',
         isRunning: false
       });
     }
   }