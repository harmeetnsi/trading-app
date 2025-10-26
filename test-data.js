const axios = require('axios');

// --- 1. CONFIGURE THIS ---
const OPENALGO_URL = 'https://openalgo.mywire.org'; 
// (Paste the API Key from your .env file)
const OPENALGO_API_KEY = 'YOUR_API_KEY_HERE'; 
// ---

async function testSmartOrder() {
  console.log(`Attempting to POST to ${OPENALGO_URL}/api/v1/smartorder...`);

  const endpoint = `${OPENALGO_URL}/api/v1/smartorder`;

  // This is the command our Go app is sending
  const requestBody = {
    apikey: OPENALGO_API_KEY,
    strategy: "Test",
    exchange: "NSE",
    symbol: "RELIANCE",
    action: "BUY",
    quantity: 1,
    product: "MIS",      // Intraday
    pricetype: "MARKET"  // Market order
  };

  try {
    const response = await axios.post(endpoint, requestBody);

    console.log('--- SUCCESS! ---');
    console.log('The endpoint /api/v1/smartorder EXISTS.');
    console.log('OpenAlgo Response:', response.data);

  } catch (error) {
    console.error('--- ERROR ---');
    if (error.response) {
      console.error(`Error Status: ${error.response.status}`);
      // We are checking if the error data is HTML (the 404 page)
      if (typeof error.response.data === 'string' && error.response.data.includes("</html>")) {
         console.error('Error Data: [HTML 404 Page]');
         console.error('---');
         console.error('CONFIRMED: The 404 error is from OpenAlgo. The endpoint /api/v1/smartorder does not exist.');
         console.error('---');
      } else {
         console.error('Error Data:', error.response.data);
         console.error('This is good news! The endpoint EXISTS, but we sent bad data (e.g., 400, 401).');
      }
    } else {
      console.error('Error:', error.message);
    }
    console.error('---');
  }
}

testSmartOrder();