// Health check endpoint for Cloud Run
export const GET = () => {
  return new Response('healthy\n', {
    status: 200,
    headers: {
      'Content-Type': 'text/plain'
    }
  });
};