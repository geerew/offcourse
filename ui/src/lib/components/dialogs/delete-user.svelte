<script lang="ts">
	import type { APIError } from '$lib/api-error.svelte';
	import { DeleteSelf } from '$lib/api/self-api';
	import { DeleteUser } from '$lib/api/user-api';
	import { auth } from '$lib/auth.svelte';
	import { Spinner } from '$lib/components';
	import { Button, Dialog, Drawer, PasswordInput } from '$lib/components/ui';
	import type { SelfDeleteModel, UserModel, UsersModel } from '$lib/models/user-model';
	import { remCalc } from '$lib/utils';
	import { Separator } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { innerWidth } from 'svelte/reactivity/window';
	import theme from 'tailwindcss/defaultTheme';

	type Props = {
		open?: boolean;
		trigger?: Snippet;
		value: UserModel | UsersModel;
		successFn?: () => void;
	};

	let { open = $bindable(false), value, trigger, successFn }: Props = $props();

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	let currentInputEl = $state<HTMLInputElement>();
	let currentPassword = $state('');
	let isPosting = $state(false);

	const isArray = Array.isArray(value);
	const deletingSelf = isArray ? false : value.id === auth?.user?.id;

	const mdBreakpoint = +theme.screens.md.replace('rem', '');
	let isDesktop = $derived(remCalc(innerWidth.current ?? 0) > mdBreakpoint);

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	$effect(() => {
		if (open) {
			currentPassword = '';
			isPosting = false;
		}
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	async function doDelete(): Promise<void> {
		isPosting = true;

		try {
			if (isArray) {
				await Promise.all(Object.values(value).map((u) => DeleteUser(u.id)));
				toast.success('Selected users deleted');
			} else {
				if (deletingSelf) {
					await DeleteSelf({ currentPassword } satisfies SelfDeleteModel);
					auth.empty();
					window.location.href = '/auth/login';
				} else {
					await DeleteUser(value.id);
				}
			}

			successFn?.();
		} catch (error) {
			toast.error((error as APIError).message);
		}

		isPosting = false;
		open = false;
	}
</script>

{#snippet alertContents()}
	<Dialog.Alert>
		<div class="text-foreground-alt-1 flex flex-col gap-2 text-center">
			{#if deletingSelf}
				<span class="text-lg">Are you sure you want to delete your account?</span>
			{:else if isArray && Object.values(value).length > 1}
				<span class="text-lg"> Are you sure you want to delete these users? </span>
			{:else}
				<span class="text-lg">Are you sure you want to delete this user?</span>
			{/if}
			<span class="text-foreground-alt-3">All associated data will be deleted</span>
		</div>

		{#if deletingSelf}
			<Separator.Root class="bg-background-alt-3 mt-2 h-px w-full shrink-0" />

			<div class="flex w-full justify-center">
				<div class="flex w-xs flex-col gap-2.5 px-2.5">
					<div>Confirm Password:</div>
					<PasswordInput
						bind:ref={currentInputEl}
						bind:value={currentPassword}
						name="current password"
					/>
				</div>
			</div>
		{/if}
	</Dialog.Alert>
{/snippet}

{#snippet deleteButton()}
	<Button
		variant="destructive"
		class="w-24"
		disabled={(deletingSelf && !currentPassword) || isPosting}
		onclick={doDelete}
	>
		{#if isPosting}
			<Spinner class="bg-foreground-alt-1 size-2" />
		{:else}
			Delete
		{/if}
	</Button>
{/snippet}

{#if isDesktop}
	<Dialog.Root bind:open {trigger}>
		<Dialog.Content
			interactOutsideBehavior="close"
			onOpenAutoFocus={(e) => {
				e.preventDefault();
				currentInputEl?.focus();
			}}
			onCloseAutoFocus={(e) => {
				e.preventDefault();
			}}
			class="w-lg"
		>
			<div class="bg-background-alt-1 overflow-hidden rounded-lg">
				{@render alertContents()}

				<Dialog.Footer>
					<Dialog.CloseButton>Cancel</Dialog.CloseButton>
					{@render deleteButton()}
				</Dialog.Footer>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{:else}
	<Drawer.Root bind:open>
		{@render trigger?.()}

		<Drawer.Content class="bg-background-alt-1" handleClass="bg-background-alt-4">
			<div class="bg-background-alt-1 overflow-hidden rounded-lg">
				{@render alertContents()}

				<Drawer.Footer>
					<Drawer.CloseButton>Cancel</Drawer.CloseButton>
					{@render deleteButton()}
				</Drawer.Footer>
			</div>
		</Drawer.Content>
	</Drawer.Root>
{/if}
