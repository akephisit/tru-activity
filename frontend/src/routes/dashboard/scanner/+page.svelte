<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '$lib/components/ui/select';
  import { Badge } from '$lib/components/ui/badge';
  import { Alert, AlertDescription } from '$lib/components/ui/alert';
  import { 
    Camera, 
    Scan, 
    CheckCircle, 
    XCircle, 
    AlertCircle,
    Users,
    Clock,
    MapPin,
    RefreshCw
  } from 'lucide-svelte';

  interface Activity {
    id: string;
    title: string;
    status: string;
    startDate: string;
    endDate: string;
    location: string;
    qrCodeRequired: boolean;
  }

  interface QRScanResult {
    success: boolean;
    message: string;
    participation?: {
      id: string;
      user: {
        firstName: string;
        lastName: string;
        studentID: string;
      };
      status: string;
      attendedAt: string;
    };
    scanLog?: {
      id: string;
      scanTimestamp: string;
    };
  }

  let activities: Activity[] = [];
  let selectedActivityId = '';
  let scanLocation = '';
  let qrInput = '';
  let scanning = false;
  let scanResult: QRScanResult | null = null;
  let error = '';
  let cameraStream: MediaStream | null = null;
  let videoElement: HTMLVideoElement;
  let canvasElement: HTMLCanvasElement;
  let scanInterval: ReturnType<typeof setInterval>;

  // Camera scanning variables
  let cameraActive = false;
  let supportsCameraAPI = false;

  const MY_ACTIVITIES_QUERY = gql`
    query MyActivityAssignments {
      myActivityAssignments {
        id
        activity {
          id
          title
          status
          startDate
          endDate
          location
          qrCodeRequired
        }
        canScanQR
      }
    }
  `;

  const SCAN_QR_MUTATION = gql`
    mutation ScanQRCode($input: QRScanInput!) {
      scanQRCode(input: $input) {
        success
        message
        participation {
          id
          user {
            firstName
            lastName
            studentID
          }
          status
          attendedAt
        }
        scanLog {
          id
          scanTimestamp
        }
      }
    }
  `;

  onMount(async () => {
    await loadActivities();
    checkCameraSupport();
  });

  onDestroy(() => {
    stopCamera();
    if (scanInterval) {
      clearInterval(scanInterval);
    }
  });

  async function loadActivities() {
    try {
      const result = await client.query({
        query: MY_ACTIVITIES_QUERY,
        fetchPolicy: 'network-only'
      });

      activities = result.data.myActivityAssignments
        .filter((assignment: any) => assignment.canScanQR)
        .map((assignment: any) => assignment.activity)
        .filter((activity: any) => activity.qrCodeRequired && activity.status === 'ACTIVE');
    } catch (err: any) {
      error = err.message || 'Failed to load activities';
      console.error('Activities error:', err);
    }
  }

  function checkCameraSupport() {
    supportsCameraAPI = !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
  }

  async function startCamera() {
    if (!supportsCameraAPI) {
      error = 'Camera not supported on this device';
      return;
    }

    try {
      cameraStream = await navigator.mediaDevices.getUserMedia({
        video: { 
          facingMode: 'environment', // Use back camera
          width: { ideal: 1280 },
          height: { ideal: 720 }
        }
      });
      
      if (videoElement) {
        videoElement.srcObject = cameraStream;
        videoElement.play();
        cameraActive = true;
        
        // Start scanning for QR codes
        scanInterval = setInterval(scanForQRCode, 1000);
      }
    } catch (err: any) {
      error = 'Failed to access camera: ' + err.message;
      console.error('Camera error:', err);
    }
  }

  function stopCamera() {
    if (cameraStream) {
      cameraStream.getTracks().forEach(track => track.stop());
      cameraStream = null;
    }
    cameraActive = false;
    
    if (scanInterval) {
      clearInterval(scanInterval);
    }
  }

  function scanForQRCode() {
    if (!videoElement || !canvasElement || !cameraActive) return;

    const context = canvasElement.getContext('2d');
    if (!context) return;

    // Draw video frame to canvas
    canvasElement.width = videoElement.videoWidth;
    canvasElement.height = videoElement.videoHeight;
    context.drawImage(videoElement, 0, 0);

    // Get image data for QR scanning
    const imageData = context.getImageData(0, 0, canvasElement.width, canvasElement.height);
    
    // Note: In a real implementation, you would use a QR code scanning library like jsQR
    // For now, we'll just show the manual input method
  }

  async function scanQRCode(qrData?: string) {
    if (!selectedActivityId) {
      error = 'Please select an activity first';
      return;
    }

    const dataToScan = qrData || qrInput;
    if (!dataToScan.trim()) {
      error = 'Please enter QR code data or scan a code';
      return;
    }

    try {
      scanning = true;
      error = '';
      scanResult = null;

      const result = await client.mutate({
        mutation: SCAN_QR_MUTATION,
        variables: {
          input: {
            qrData: dataToScan,
            activityID: selectedActivityId,
            scanLocation: scanLocation || 'Mobile Scanner'
          }
        }
      });

      scanResult = result.data.scanQRCode;
      
      if (scanResult?.success) {
        // Clear input for next scan
        qrInput = '';
        // Optional: Play success sound or vibrate
        if (navigator.vibrate) {
          navigator.vibrate(200);
        }
      }
    } catch (err: any) {
      error = err.message || 'Failed to scan QR code';
      console.error('Scan error:', err);
    } finally {
      scanning = false;
    }
  }

  function clearResult() {
    scanResult = null;
    error = '';
  }

  function getStatusColor(status: string) {
    switch (status.toLowerCase()) {
      case 'attended': return 'text-green-600 bg-green-50';
      case 'approved': return 'text-blue-600 bg-blue-50';
      case 'pending': return 'text-yellow-600 bg-yellow-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  }

  $: selectedActivity = activities.find(a => a.id === selectedActivityId);
</script>

<div class="container mx-auto py-4 px-4 max-w-md">
  <div class="space-y-4">
    <!-- Header -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2">
          <Scan size={20} />
          QR Code Scanner
        </CardTitle>
      </CardHeader>
    </Card>

    <!-- Activity Selection -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-base">Select Activity</CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <div>
          <Label for="activity">Activity</Label>
          <Select type="single" onValueChange={(value: string | string[]) => selectedActivityId = Array.isArray(value) ? value[0] : value}>
            <SelectTrigger>
              <SelectValue placeholder="Choose an activity to scan for" />
            </SelectTrigger>
            <SelectContent>
              {#each activities as activity}
                <SelectItem value={activity.id}>
                  {activity.title}
                </SelectItem>
              {/each}
            </SelectContent>
          </Select>
        </div>

        {#if selectedActivity}
          <div class="p-3 bg-blue-50 rounded-lg space-y-2">
            <div class="flex items-center gap-2 text-sm">
              <Clock size={14} />
              <span>{new Date(selectedActivity.startDate).toLocaleString()}</span>
            </div>
            {#if selectedActivity.location}
              <div class="flex items-center gap-2 text-sm">
                <MapPin size={14} />
                <span>{selectedActivity.location}</span>
              </div>
            {/if}
            <Badge variant="outline" class="text-xs">
              {selectedActivity.status}
            </Badge>
          </div>
        {/if}

        <div>
          <Label for="location">Scan Location (Optional)</Label>
          <Input
            id="location"
            bind:value={scanLocation}
            placeholder="e.g., Room A101, Main Entrance"
          />
        </div>
      </CardContent>
    </Card>

    <!-- Camera Scanner -->
    {#if supportsCameraAPI}
      <Card>
        <CardHeader class="pb-3">
          <CardTitle class="text-base flex items-center justify-between">
            <span>Camera Scanner</span>
            <Button 
              variant="outline" 
              size="sm" 
              onclick={cameraActive ? stopCamera : startCamera}
            >
              <Camera size={16} class="mr-2" />
              {cameraActive ? 'Stop' : 'Start'} Camera
            </Button>
          </CardTitle>
        </CardHeader>
        {#if cameraActive}
          <CardContent class="p-0">
            <div class="relative">
              <video 
                bind:this={videoElement}
                class="w-full h-64 object-cover rounded-lg"
                autoplay
                muted
                playsinline
              ></video>
              <canvas bind:this={canvasElement} class="hidden"></canvas>
              
              <!-- Scanning overlay -->
              <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div class="w-48 h-48 border-2 border-blue-500 rounded-lg relative">
                  <div class="absolute top-0 left-0 w-6 h-6 border-t-4 border-l-4 border-blue-500 rounded-tl-lg"></div>
                  <div class="absolute top-0 right-0 w-6 h-6 border-t-4 border-r-4 border-blue-500 rounded-tr-lg"></div>
                  <div class="absolute bottom-0 left-0 w-6 h-6 border-b-4 border-l-4 border-blue-500 rounded-bl-lg"></div>
                  <div class="absolute bottom-0 right-0 w-6 h-6 border-b-4 border-r-4 border-blue-500 rounded-br-lg"></div>
                </div>
              </div>
            </div>
            <div class="p-4 text-center text-sm text-gray-600">
              Position QR code within the frame to scan automatically
            </div>
          </CardContent>
        {/if}
      </Card>
    {/if}

    <!-- Manual Input -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-base">Manual Input</CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <div>
          <Label for="qr-input">QR Code Data</Label>
          <Input
            id="qr-input"
            bind:value={qrInput}
            placeholder="Paste or type QR code data here"
            disabled={scanning}
          />
        </div>
        
        <Button 
          onclick={() => scanQRCode()} 
          disabled={scanning || !selectedActivityId || !qrInput.trim()}
          class="w-full"
        >
          {#if scanning}
            <RefreshCw size={16} class="mr-2 animate-spin" />
            Scanning...
          {:else}
            <Scan size={16} class="mr-2" />
            Scan QR Code
          {/if}
        </Button>
      </CardContent>
    </Card>

    <!-- Error Display -->
    {#if error}
      <Alert variant="destructive">
        <AlertCircle size={16} />
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    {/if}

    <!-- Scan Result -->
    {#if scanResult}
      <Card class="border-2 {scanResult.success ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'}">
        <CardHeader class="pb-3">
          <CardTitle class="text-base flex items-center gap-2">
            {#if scanResult.success}
              <CheckCircle size={20} class="text-green-600" />
              <span class="text-green-600">Scan Successful</span>
            {:else}
              <XCircle size={20} class="text-red-600" />
              <span class="text-red-600">Scan Failed</span>
            {/if}
          </CardTitle>
        </CardHeader>
        <CardContent class="space-y-3">
          <p class="text-sm">{scanResult.message}</p>
          
          {#if scanResult.participation}
            <div class="p-3 bg-white rounded border">
              <div class="flex items-center justify-between mb-2">
                <span class="font-medium">
                  {scanResult.participation.user.firstName} {scanResult.participation.user.lastName}
                </span>
                <Badge variant="outline" class={getStatusColor(scanResult.participation.status)}>
                  {scanResult.participation.status}
                </Badge>
              </div>
              <div class="text-sm text-gray-600">
                <div>Student ID: {scanResult.participation.user.studentID}</div>
                {#if scanResult.participation.attendedAt}
                  <div>Attended: {new Date(scanResult.participation.attendedAt).toLocaleString()}</div>
                {/if}
              </div>
            </div>
          {/if}

          <Button variant="outline" size="sm" onclick={clearResult} class="w-full">
            Scan Next
          </Button>
        </CardContent>
      </Card>
    {/if}

    <!-- Instructions -->
    <Card>
      <CardContent class="pt-6">
        <div class="text-sm text-gray-600 space-y-2">
          <h4 class="font-medium">How to use:</h4>
          <ol class="list-decimal list-inside space-y-1">
            <li>Select the activity you want to scan for</li>
            <li>Use camera scanner or paste QR code data manually</li>
            <li>Confirm scan results and attendance</li>
          </ol>
        </div>
      </CardContent>
    </Card>
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
</style>