<script lang="ts">
	import { client } from '$lib/graphql/client';
	import { GET_ME } from '$lib/graphql/queries';
	import { authStore, isAuthenticated, user, isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';
	import { Separator } from '$lib/components/ui/separator';
	import { 
		DropdownMenu, 
		DropdownMenuContent, 
		DropdownMenuItem, 
		DropdownMenuSeparator, 
		DropdownMenuTrigger 
	} from '$lib/components/ui/dropdown-menu';
	import { 
		Sidebar,
		SidebarContent,
		SidebarHeader,
		SidebarMenu,
		SidebarMenuItem,
		SidebarMenuButton,
		SidebarProvider,
		SidebarInset
	} from '$lib/components/ui/sidebar';
	import { 
		Home, 
		Calendar, 
		Users, 
		Settings, 
		LogOut, 
		Building2, 
		GraduationCap,
		UserCheck,
		BarChart3
	} from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import NotificationCenter from '$lib/components/NotificationCenter.svelte';

	let { children } = $props();

	// Redirect if not authenticated
	$effect(() => {
		if (!$isAuthenticated) {
			goto('/login');
		}
	});

	// Load user data on mount if authenticated
	$effect(() => {
		if ($isAuthenticated && !$user) {
			loadUserData();
		}
	});

	async function loadUserData() {
		try {
			const result = await client.query({
				query: GET_ME
			});
			if (result.data?.me) {
				authStore.updateUser(result.data.me);
			}
		} catch (error) {
			console.error('Failed to load user data:', error);
			// If query fails due to auth, logout
			authStore.logout();
			goto('/login');
		}
	}

	function handleLogout() {
		authStore.logout();
		toast.success('ออกจากระบบเรียบร้อยแล้ว');
		goto('/login');
	}

	// Navigation items based on user role
	const navigationItems = $derived([
		{
			title: 'หน้าหลัก',
			href: '/dashboard',
			icon: Home,
			roles: ['STUDENT', 'SUPER_ADMIN', 'FACULTY_ADMIN', 'REGULAR_ADMIN']
		},
		{
			title: 'แดชบอร์ดผู้ดูแลระบบ',
			href: '/dashboard/admin',
			icon: BarChart3,
			roles: ['SUPER_ADMIN']
		},
		{
			title: 'แดชบอร์ดคณะ',
			href: '/dashboard/faculty-admin',
			icon: Building2,
			roles: ['FACULTY_ADMIN']
		},
		{
			title: 'กิจกรรม',
			href: '/dashboard/activities',
			icon: Calendar,
			roles: ['STUDENT', 'SUPER_ADMIN', 'FACULTY_ADMIN', 'REGULAR_ADMIN']
		},
		{
			title: 'กิจกรรมของฉัน',
			href: '/dashboard/my-activities',
			icon: UserCheck,
			roles: ['STUDENT']
		},
		{
			title: 'QR Scanner',
			href: '/dashboard/scanner',
			icon: UserCheck,
			roles: ['REGULAR_ADMIN']
		},
		{
			title: 'My QR Code',
			href: '/dashboard/my-qr',
			icon: UserCheck,
			roles: ['STUDENT']
		},
		{
			title: 'จัดการกิจกรรม',
			href: '/dashboard/manage-activities',
			icon: Settings,
			roles: ['SUPER_ADMIN', 'FACULTY_ADMIN', 'REGULAR_ADMIN']
		},
		{
			title: 'จัดการผู้ใช้',
			href: '/dashboard/users',
			icon: Users,
			roles: ['SUPER_ADMIN', 'FACULTY_ADMIN']
		},
		{
			title: 'จัดการคณะ',
			href: '/dashboard/faculties',
			icon: Building2,
			roles: ['SUPER_ADMIN']
		},
		{
			title: 'จัดการภาควิชา',
			href: '/dashboard/departments',
			icon: GraduationCap,
			roles: ['SUPER_ADMIN', 'FACULTY_ADMIN']
		},
		{
			title: 'รายงาน',
			href: '/dashboard/reports',
			icon: BarChart3,
			roles: ['SUPER_ADMIN', 'FACULTY_ADMIN', 'REGULAR_ADMIN']
		}
	]);

	const filteredNavigation = $derived(navigationItems.filter(item => 
		item.roles.includes($user?.role || 'STUDENT')
	));
</script>

{#if $isAuthenticated}
	<SidebarProvider>
		<Sidebar>
			<SidebarHeader>
				<div class="flex items-center gap-2 px-4 py-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
						<GraduationCap class="h-4 w-4" />
					</div>
					<div class="grid flex-1 text-left text-sm leading-tight">
						<span class="truncate font-semibold">TRU Activity</span>
						<span class="truncate text-xs text-muted-foreground">
							{$user?.faculty?.name || 'ระบบเก็บกิจกรรม'}
						</span>
					</div>
				</div>
			</SidebarHeader>
			
			<SidebarContent>
				<SidebarMenu>
					{#each filteredNavigation as item}
						<SidebarMenuItem>
							<SidebarMenuButton isActive={$page.url.pathname === item.href}>
								{#snippet child({ props })}
									<a href={item.href} {...props}>
										<item.icon class="h-4 w-4" />
										<span>{item.title}</span>
									</a>
								{/snippet}
							</SidebarMenuButton>
						</SidebarMenuItem>
					{/each}
				</SidebarMenu>
			</SidebarContent>
		</Sidebar>

		<SidebarInset>
			<!-- Header -->
			<header class="flex h-16 shrink-0 items-center gap-2 border-b px-4">
				<div class="flex flex-1 items-center gap-2">
					<h1 class="text-lg font-semibold">TRU Activity Dashboard</h1>
				</div>
				
				<div class="flex items-center gap-2">
					{#if $user}
						<DropdownMenu>
							<DropdownMenuTrigger>
								{#snippet child({ props })}
									<Button {...props} variant="ghost" size="sm" class="h-8 w-8 rounded-full">
										<Avatar class="h-8 w-8">
											<AvatarFallback class="text-xs">
												{$user.firstName.charAt(0)}{$user.lastName.charAt(0)}
											</AvatarFallback>
										</Avatar>
									</Button>
								{/snippet}
							</DropdownMenuTrigger>
							<DropdownMenuContent align="end" class="w-56">
								<div class="flex items-center justify-start gap-2 p-2">
									<div class="flex flex-col space-y-1 leading-none">
										<p class="font-medium">{$user.firstName} {$user.lastName}</p>
										<p class="w-[200px] truncate text-sm text-muted-foreground">
											{$user.email}
										</p>
										<p class="text-xs text-muted-foreground">
											{$user.role === 'STUDENT' ? 'นักศึกษา' : 
											 $user.role === 'SUPER_ADMIN' ? 'ผู้ดูแลระบบ' :
											 $user.role === 'FACULTY_ADMIN' ? 'ผู้ดูแลคณะ' : 'ผู้ดูแล'} 
											({$user.studentID})
										</p>
									</div>
								</div>
								<DropdownMenuSeparator />
								<DropdownMenuItem onclick={handleLogout}>
									<LogOut class="mr-2 h-4 w-4" />
									<span>ออกจากระบบ</span>
								</DropdownMenuItem>
							</DropdownMenuContent>
						</DropdownMenu>
					{/if}
				</div>
			</header>

			<!-- Main content -->
			<main class="flex-1 overflow-auto p-4">
				{@render children()}
			</main>
		</SidebarInset>
	</SidebarProvider>

	<!-- Real-time Notification Center -->
	<NotificationCenter position="top-right" enableSound={true} />
{/if}