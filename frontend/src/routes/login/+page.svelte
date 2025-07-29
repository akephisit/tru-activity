<script lang="ts">
	import { mutation } from '@apollo/client';
	import { LOGIN_MUTATION } from '$lib/graphql/mutations';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { toast } from 'svelte-sonner';

	let email = $state('');
	let password = $state('');
	let isLoading = $state(false);

	const login = mutation(LOGIN_MUTATION);

	async function handleLogin() {
		if (!email || !password) {
			toast.error('กรุณากรอกอีเมลและรหัสผ่าน');
			return;
		}

		isLoading = true;
		
		try {
			const result = await login({
				variables: {
					input: {
						email,
						password
					}
				}
			});

			if (result.data?.login) {
				const { token, user } = result.data.login;
				authStore.login(token, user);
				toast.success(`ยินดีต้อนรับ ${user.firstName} ${user.lastName}`);
				goto('/dashboard');
			}
		} catch (error: any) {
			console.error('Login error:', error);
			toast.error(error.message || 'เข้าสู่ระบบไม่สำเร็จ');
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
				เข้าสู่ระบบ TRU Activity
			</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี
			</p>
		</div>
		
		<Card>
			<CardHeader>
				<CardTitle>เข้าสู่ระบบ</CardTitle>
				<CardDescription>กรุณากรอกข้อมูลของคุณเพื่อเข้าสู่ระบบ</CardDescription>
			</CardHeader>
			<CardContent>
				<form onsubmit|preventDefault={handleLogin} class="space-y-6">
					<div>
						<Label for="email">อีเมล</Label>
						<Input
							id="email"
							type="email"
							bind:value={email}
							placeholder="student@mail.rmutt.ac.th"
							required
							disabled={isLoading}
						/>
					</div>
					
					<div>
						<Label for="password">รหัสผ่าน</Label>
						<Input
							id="password"
							type="password"
							bind:value={password}
							placeholder="รหัสผ่าน"
							required
							disabled={isLoading}
						/>
					</div>

					<Button type="submit" class="w-full" disabled={isLoading}>
						{isLoading ? 'กำลังเข้าสู่ระบบ...' : 'เข้าสู่ระบบ'}
					</Button>
				</form>

				<div class="mt-6">
					<div class="relative">
						<div class="absolute inset-0 flex items-center">
							<div class="w-full border-t border-gray-300" />
						</div>
						<div class="relative flex justify-center text-sm">
							<span class="px-2 bg-white text-gray-500">หรือ</span>
						</div>
					</div>

					<div class="mt-6">
						<Button variant="outline" class="w-full" onclick={() => goto('/register')}>
							สมัครสมาชิกใหม่
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>
	</div>
</div>