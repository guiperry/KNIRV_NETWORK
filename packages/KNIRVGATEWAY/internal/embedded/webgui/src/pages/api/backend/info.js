import axios from 'axios';

   export default async function handler(req, res) {
     if (req.method !== 'GET') {
       return res.status(405).json({ message: 'Method not allowed' });
     }

     try {
       const response = await axios.get(`${process.env.NEXT_PUBLIC_BACKEND_URL}/info`);
       return res.status(200).json(response.data);
     } catch (error) {
       console.error('Error fetching backend info:', error);
       return res.status(500).json({ message: 'Failed to fetch backend info' });
     }
   }