<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Badge } from '$lib/components/ui/badge';
  import { Alert, AlertDescription } from '$lib/components/ui/alert';
  import { 
    QrCode, 
    RefreshCw, 
    Download, 
    Copy, 
    CheckCircle,
    AlertCircle,
    Clock,
    Shield
  } from 'lucide-svelte';

  interface QRData {
    studentID: string;
    timestamp: string;
    signature: string;
    version: number;
    qrString: string;
  }

  let qrData: QRData | null = null;
  let loading = false;
  let error = '';
  let copied = false;
  let qrCodeDataURL = '';

  const MY_QR_QUERY = gql`
    query MyQRData {
      myQRData {
        studentID
        timestamp
        signature
        version
        qrString
      }
    }
  `;

  const REFRESH_QR_MUTATION = gql`
    mutation RefreshMyQRSecret {
      refreshMyQRSecret {
        studentID
        timestamp
        signature
        version
        qrString
      }
    }
  `;

  onMount(async () => {
    await loadQRData();
  });

  async function loadQRData() {
    try {
      loading = true;
      error = '';

      const result = await client.query({
        query: MY_QR_QUERY,
        fetchPolicy: 'network-only'
      });

      qrData = result.data.myQRData;
      await generateQRCodeImage();
    } catch (err: any) {
      error = err.message || 'Failed to load QR data';
      console.error('QR data error:', err);
    } finally {
      loading = false;
    }
  }

  async function refreshQRSecret() {
    if (!confirm('Are you sure you want to refresh your QR secret? Your old QR codes will no longer work.')) {
      return;
    }

    try {
      loading = true;
      error = '';

      const result = await client.mutate({
        mutation: REFRESH_QR_MUTATION
      });

      qrData = result.data.refreshMyQRSecret;
      await generateQRCodeImage();
    } catch (err: any) {
      error = err.message || 'Failed to refresh QR secret';
      console.error('QR refresh error:', err);
    } finally {
      loading = false;
    }
  }

  async function generateQRCodeImage() {
    if (!qrData) return;

    try {
      // Use QRCode.js library (you'll need to install: npm install qrcode)
      // For now, we'll create a simple data URL representation
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      
      if (ctx) {
        canvas.width = 300;
        canvas.height = 300;
        
        // Simple placeholder - in real implementation, use QRCode library
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(0, 0, 300, 300);
        ctx.fillStyle = '#000000';
        ctx.font = '12px monospace';
        ctx.textAlign = 'center';
        ctx.fillText('QR Code', 150, 150);
        ctx.fillText(qrData.studentID, 150, 170);
        
        qrCodeDataURL = canvas.toDataURL();
      }
    } catch (err) {
      console.error('Failed to generate QR code image:', err);
    }
  }

  async function copyQRData() {
    if (!qrData) return;

    try {
      await navigator.clipboard.writeText(qrData.qrString);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    } catch (err) {
      console.error('Failed to copy QR data:', err);
    }
  }

  function downloadQRCode() {
    if (!qrCodeDataURL) return;

    const link = document.createElement('a');
    link.download = `my-qr-code-${Date.now()}.png`;
    link.href = qrCodeDataURL;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  function getTimeRemaining() {
    if (!qrData) return '';
    
    const qrTime = parseInt(qrData.timestamp) * 1000;
    const expiryTime = qrTime + (15 * 60 * 1000); // 15 minutes expiry
    const now = Date.now();
    const remaining = expiryTime - now;
    
    if (remaining <= 0) {
      return 'Expired';
    }
    
    const minutes = Math.floor(remaining / (60 * 1000));
    const seconds = Math.floor((remaining % (60 * 1000)) / 1000);
    
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }

  function isQRExpired() {
    if (!qrData) return true;
    
    const qrTime = parseInt(qrData.timestamp) * 1000;
    const expiryTime = qrTime + (15 * 60 * 1000); // 15 minutes expiry
    
    return Date.now() > expiryTime;
  }

  // Auto-refresh QR data every minute to keep it fresh
  let refreshInterval: ReturnType<typeof setInterval>;
  onMount(() => {
    refreshInterval = setInterval(() => {
      if (isQRExpired()) {
        loadQRData();
      }
    }, 60000);
    
    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  });
</script>

<div class="container mx-auto py-6 px-4 max-w-lg">
  <div class="space-y-6">
    <!-- Header -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2">
          <QrCode size={24} />
          My QR Code
        </CardTitle>
        <p class="text-sm text-gray-600">
          Your personal QR code for activity check-ins
        </p>
      </CardHeader>
    </Card>

    {#if error}
      <Alert variant="destructive">
        <AlertCircle size={16} />
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    {/if}

    {#if loading}
      <Card>
        <CardContent class="pt-6 text-center">
          <RefreshCw size={32} class="mx-auto animate-spin text-gray-400 mb-4" />
          <p class="text-gray-600">Loading your QR code...</p>
        </CardContent>
      </Card>
    {:else if qrData}
      <!-- QR Code Display -->
      <Card class="relative">
        <CardHeader class="pb-3">
          <div class="flex items-center justify-between">
            <CardTitle class="text-lg">Current QR Code</CardTitle>
            <div class="flex items-center gap-2">
              <Badge variant={isQRExpired() ? 'destructive' : 'default'} class="text-xs">
                <Clock size={12} class="mr-1" />
                {getTimeRemaining()}
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <!-- QR Code Image -->
          <div class="flex justify-center">
            <div class="p-4 bg-white rounded-lg border-2 border-gray-200 shadow-sm">
              {#if qrCodeDataURL}
                <img 
                  src={qrCodeDataURL} 
                  alt="Your QR Code" 
                  class="w-48 h-48 {isQRExpired() ? 'opacity-50' : ''}"
                />
              {:else}
                <div class="w-48 h-48 bg-gray-100 flex items-center justify-center rounded">
                  <QrCode size={64} class="text-gray-400" />
                </div>
              {/if}
            </div>
          </div>

          <!-- QR Info -->
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-gray-600">Student ID:</span>
              <span class="font-medium">{qrData.studentID}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-600">Generated:</span>
              <span class="font-medium">
                {new Date(parseInt(qrData.timestamp) * 1000).toLocaleString()}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-600">Version:</span>
              <span class="font-medium">{qrData.version}</span>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="grid grid-cols-2 gap-3">
            <Button variant="outline" onclick={copyQRData} disabled={isQRExpired()}>
              {#if copied}
                <CheckCircle size={16} class="mr-2 text-green-600" />
                Copied!
              {:else}
                <Copy size={16} class="mr-2" />
                Copy Data
              {/if}
            </Button>
            <Button variant="outline" onclick={downloadQRCode} disabled={!qrCodeDataURL || isQRExpired()}>
              <Download size={16} class="mr-2" />
              Download
            </Button>
          </div>

          {#if isQRExpired()}
            <Alert variant="destructive">
              <AlertCircle size={16} />
              <AlertDescription>
                Your QR code has expired. Please generate a new one.
              </AlertDescription>
            </Alert>
          {/if}
        </CardContent>
      </Card>

      <!-- Refresh Section -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base flex items-center gap-2">
            <Shield size={20} />
            Security Settings
          </CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <p class="text-sm text-gray-600">
            Your QR code is automatically refreshed every 15 minutes for security. 
            You can also manually refresh your secret key if needed.
          </p>
          
          <Button 
            variant="outline" 
            onclick={refreshQRSecret} 
            disabled={loading}
            class="w-full"
          >
            <RefreshCw size={16} class="mr-2" />
            Refresh Secret Key
          </Button>
          
          <div class="text-xs text-gray-500 space-y-1">
            <p>• QR codes expire after 15 minutes</p>
            <p>• Refreshing your secret invalidates all old QR codes</p>
            <p>• Only use this feature if your QR code is compromised</p>
          </div>
        </CardContent>
      </Card>

      <!-- Instructions -->
      <Card>
        <CardContent class="pt-6">
          <div class="text-sm text-gray-600 space-y-3">
            <h4 class="font-medium text-gray-900">How to use your QR code:</h4>
            <ol class="list-decimal list-inside space-y-2">
              <li>Show this QR code to activity administrators for check-in</li>
              <li>Make sure your QR code is not expired (check the timer)</li>
              <li>Keep your phone screen bright for easy scanning</li>
              <li>You can download the QR code to save it offline</li>
            </ol>
            
            <div class="mt-4 p-3 bg-blue-50 rounded-lg">
              <h5 class="font-medium text-blue-900 mb-1">Security Note</h5>
              <p class="text-blue-700 text-xs">
                Your QR code contains encrypted data that verifies your identity. 
                Never share your QR code data with others, and report any suspicious activity.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    {/if}
  </div>
</div>

<style>
  /* Mobile-optimized styles */
  @media (max-width: 640px) {
    .container {
      padding-left: 0.5rem;
      padding-right: 0.5rem;
    }
  }

  /* QR code animation for non-expired codes */
  img:not(.opacity-50) {
    animation: subtle-pulse 2s ease-in-out infinite;
  }

  @keyframes subtle-pulse {
    0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
    50% { box-shadow: 0 0 0 8px rgba(59, 130, 246, 0); }
  }
</style>