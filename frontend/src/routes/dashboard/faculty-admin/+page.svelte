<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { user } from '$lib/stores/auth';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { 
    Users, 
    Calendar, 
    GraduationCap, 
    TrendingUp, 
    AlertTriangle,
    Plus,
    Settings,
    BarChart3,
    Clock
  } from 'lucide-svelte';
  import { toast } from 'svelte-sonner';
  import { goto } from '$app/navigation';

  interface FacultyStats {
    totalStudents: number;
    activeStudents: number;
    totalActivities: number;
    activeActivities: number;
    totalParticipations: number;
    averageAttendance: number;
  }

  interface Activity {
    id: string;
    title: string;
    status: string;
    startDate: string;
    endDate: string;
    maxParticipants: number;
    currentParticipants: number;
    points: number;
  }

  interface Student {
    id: string;
    studentID: string;
    firstName: string;
    lastName: string;
    email: string;
    isActive: boolean;
    department?: {
      name: string;
    };
    totalPoints: number;
    activitiesCount: number;
  }

  interface RegularAdmin {
    id: string;
    studentID: string;
    firstName: string;
    lastName: string;
    email: string;
    isActive: boolean;
    assignedActivities: number;
  }

  let facultyStats: FacultyStats = {
    totalStudents: 0,
    activeStudents: 0,
    totalActivities: 0,  
    activeActivities: 0,
    totalParticipations: 0,
    averageAttendance: 0
  };
  let activities: Activity[] = [];
  let students: Student[] = [];
  let regularAdmins: RegularAdmin[] = [];
  let loading = true;
  let error = '';

  const FACULTY_DASHBOARD_QUERY = gql`
    query FacultyDashboard($facultyId: ID!) {
      facultyStats(facultyId: $facultyId) {
        totalStudents
        activeStudents
        totalActivities
        activeActivities
        totalParticipations
        averageAttendance
      }
      
      facultyActivities(facultyId: $facultyId, limit: 10) {
        id
        title
        status
        startDate
        endDate
        maxParticipants
        currentParticipants
        points
      }
      
      facultyStudents(facultyId: $facultyId, limit: 10) {
        id
        studentID
        firstName
        lastName
        email
        isActive
        department {
          name
        }
        totalPoints
        activitiesCount
      }
      
      facultyRegularAdmins(facultyId: $facultyId) {
        id
        studentID
        firstName
        lastName
        email
        isActive
        assignedActivities
      }
    }
  `;

  onMount(async () => {
    await loadDashboardData();
  });

  async function loadDashboardData() {
    if (!$user?.faculty?.id) {
      error = 'Faculty information not available';
      loading = false;
      return;
    }

    try {
      loading = true;
      error = '';

      const result = await client.query({
        query: FACULTY_DASHBOARD_QUERY,
        variables: {
          facultyId: $user.faculty.id
        }
      });

      facultyStats = result.data.facultyStats;
      activities = result.data.facultyActivities;
      students = result.data.facultyStudents;
      regularAdmins = result.data.facultyRegularAdmins;
    } catch (err: any) {
      error = err.message || 'Failed to load faculty dashboard data';
      console.error('Faculty dashboard error:', err);
      toast.error('ไม่สามารถโหลดข้อมูลได้');
    } finally {
      loading = false;
    }
  }

  function getStatusBadgeVariant(status: string) {
    switch (status.toLowerCase()) {
      case 'active': return 'default';
      case 'draft': return 'secondary';
      case 'completed': return 'outline';
      case 'cancelled': return 'destructive';
      default: return 'secondary';
    }
  }

  function getStatusLabel(status: string) {
    const statusMap = {
      'DRAFT': 'ร่าง',
      'ACTIVE': 'เปิดรับสมัคร',
      'COMPLETED': 'เสร็จสิ้น',
      'CANCELLED': 'ยกเลิก'
    };
    return statusMap[status as keyof typeof statusMap] || status;
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleDateString('th-TH', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  function calculateAttendanceRate() {
    if (facultyStats.totalParticipations === 0) return 0;
    return facultyStats.averageAttendance;
  }
</script>

<div class="container mx-auto py-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold">Faculty Admin Dashboard</h1>
      <p class="text-muted-foreground">
        {$user?.faculty?.name} ({$user?.faculty?.code})
      </p>
    </div>
    <div class="flex gap-2">
      <Button variant="outline" onclick={loadDashboardData} disabled={loading}>
        {loading ? 'กำลังโหลด...' : 'รีเฟรช'}
      </Button>
      <Button onclick={() => goto('/dashboard/manage-activities')}>
        <Plus class="w-4 h-4 mr-2" />
        สร้างกิจกรรม
      </Button>
    </div>
  </div>

  {#if error}
    <Card class="border-red-200 bg-red-50">
      <CardContent class="pt-6">
        <div class="flex items-center gap-2 text-red-600">
          <AlertTriangle size={20} />
          <span>{error}</span>
        </div>
      </CardContent>
    </Card>
  {/if}

  <!-- Faculty Statistics -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">นักศึกษาทั้งหมด</CardTitle>
        <GraduationCap class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{facultyStats.totalStudents}</div>
        <p class="text-xs text-muted-foreground">
          {facultyStats.activeStudents} คนที่ใช้งานอยู่
        </p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">กิจกรรมทั้งหมด</CardTitle>
        <Calendar class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{facultyStats.totalActivities}</div>
        <p class="text-xs text-muted-foreground">
          {facultyStats.activeActivities} กิจกรรมที่เปิดอยู่
        </p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">การเข้าร่วมกิจกรรม</CardTitle>
        <Users class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{facultyStats.totalParticipations}</div>
        <p class="text-xs text-muted-foreground">ครั้งทั้งหมด</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">อัตราการเข้าร่วม</CardTitle>
        <TrendingUp class="h-4 w-4 text-green-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-green-600">
          {calculateAttendanceRate().toFixed(1)}%
        </div>
        <p class="text-xs text-muted-foreground">เฉลี่ยของคณะ</p>
      </CardContent>
    </Card>
  </div>

  <div class="grid gap-6 lg:grid-cols-2">
    <!-- Recent Activities -->
    <Card>
      <CardHeader class="flex flex-row items-center justify-between">
        <div>
          <CardTitle>กิจกรรมล่าสุด</CardTitle>
          <p class="text-sm text-muted-foreground">กิจกรรมของคณะ</p>
        </div>
        <Button variant="outline" size="sm" onclick={() => goto('/dashboard/manage-activities')}>
          <Settings class="w-4 h-4 mr-2" />
          จัดการ
        </Button>
      </CardHeader>
      <CardContent>
        <div class="space-y-4">
          {#each activities.slice(0, 5) as activity}
            <div class="flex items-center justify-between p-3 border rounded-lg">
              <div class="space-y-1">
                <h4 class="font-medium">{activity.title}</h4>
                <div class="flex items-center gap-4 text-sm text-muted-foreground">
                  <span class="flex items-center gap-1">
                    <Clock class="w-3 h-3" />
                    {formatDate(activity.startDate)}
                  </span>
                  <span class="flex items-center gap-1">
                    <Users class="w-3 h-3" />
                    {activity.currentParticipants}/{activity.maxParticipants}
                  </span>
                  {#if activity.points > 0}
                    <span>{activity.points} คะแนน</span>
                  {/if}
                </div>
              </div>
              <Badge variant={getStatusBadgeVariant(activity.status)}>
                {getStatusLabel(activity.status)}
              </Badge>
            </div>
          {/each}
          {#if activities.length === 0}
            <p class="text-center text-muted-foreground py-4">
              ยังไม่มีกิจกรรม
            </p>
          {/if}
        </div>
      </CardContent>
    </Card>

    <!-- Active Students -->
    <Card>
      <CardHeader class="flex flex-row items-center justify-between">
        <div>
          <CardTitle>นักศึกษาที่ใช้งานมากที่สุด</CardTitle>
          <p class="text-sm text-muted-foreground">นักศึกษาของคณะ</p>
        </div>
        <Button variant="outline" size="sm" onclick={() => goto('/dashboard/users')}>
          <Users class="w-4 h-4 mr-2" />
          ดูทั้งหมด
        </Button>
      </CardHeader>
      <CardContent>
        <div class="space-y-4">
          {#each students.slice(0, 5) as student}
            <div class="flex items-center justify-between p-3 border rounded-lg">
              <div class="space-y-1">
                <h4 class="font-medium">{student.firstName} {student.lastName}</h4>
                <div class="flex items-center gap-4 text-sm text-muted-foreground">
                  <span>{student.studentID}</span>
                  {#if student.department}
                    <span>{student.department.name}</span>
                  {/if}
                </div>
              </div>
              <div class="text-right">
                <div class="font-medium text-green-600">{student.totalPoints} คะแนน</div>
                <div class="text-sm text-muted-foreground">{student.activitiesCount} กิจกรรม</div>
              </div>
            </div>
          {/each}
          {#if students.length === 0}
            <p class="text-center text-muted-foreground py-4">
              ไม่มีข้อมูลนักศึกษา
            </p>
          {/if}
        </div>
      </CardContent>
    </Card>
  </div>

  <!-- Regular Admins Management -->
  <Card>
    <CardHeader class="flex flex-row items-center justify-between">
      <div>
        <CardTitle>ผู้ดูแลระดับปฏิบัติการ</CardTitle>
        <p class="text-sm text-muted-foreground">Admin ที่ได้รับมอบหมายในคณะ</p>
      </div>
      <Button variant="outline" size="sm">
        <Plus class="w-4 h-4 mr-2" />
        เพิ่ม Admin
      </Button>
    </CardHeader>
    <CardContent>
      <div class="space-y-4">
        {#each regularAdmins as admin}
          <div class="flex items-center justify-between p-3 border rounded-lg">
            <div class="space-y-1">
              <h4 class="font-medium">{admin.firstName} {admin.lastName}</h4>
              <div class="flex items-center gap-4 text-sm text-muted-foreground">
                <span>{admin.studentID}</span>
                <span>{admin.email}</span>
              </div>
            </div>
            <div class="flex items-center gap-4">
              <div class="text-right">
                <div class="font-medium">{admin.assignedActivities} กิจกรรม</div>
                <div class="text-sm text-muted-foreground">ที่ได้รับมอบหมาย</div>
              </div>
              <Badge variant={admin.isActive ? 'default' : 'secondary'}>
                {admin.isActive ? 'ใช้งานอยู่' : 'ไม่ใช้งาน'}
              </Badge>
            </div>
          </div>
        {/each}
        {#if regularAdmins.length === 0}
          <p class="text-center text-muted-foreground py-4">
            ยังไม่มี Regular Admin ในคณะนี้
          </p>
        {/if}
      </div>
    </CardContent>
  </Card>

  <!-- Quick Actions -->
  <Card>
    <CardHeader>
      <CardTitle>การดำเนินการด่วน</CardTitle>
    </CardHeader>
    <CardContent>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Button variant="outline" class="h-20 flex-col" onclick={() => goto('/dashboard/manage-activities')}>
          <Calendar class="w-6 h-6 mb-2" />
          <span class="text-sm">จัดการกิจกรรม</span>
        </Button>
        <Button variant="outline" class="h-20 flex-col" onclick={() => goto('/dashboard/users')}>
          <Users class="w-6 h-6 mb-2" />
          <span class="text-sm">จัดการนักศึกษา</span>
        </Button>
        <Button variant="outline" class="h-20 flex-col" onclick={() => goto('/dashboard/reports')}>
          <BarChart3 class="w-6 h-6 mb-2" />
          <span class="text-sm">ดูรายงาน</span>
        </Button>
        <Button variant="outline" class="h-20 flex-col" onclick={() => goto('/dashboard/departments')}>
          <GraduationCap class="w-6 h-6 mb-2" />
          <span class="text-sm">จัดการภาควิชา</span>
        </Button>
      </div>
    </CardContent>
  </Card>
</div>