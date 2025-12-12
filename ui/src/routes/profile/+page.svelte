<script lang="ts">
	import type { APIError } from '$lib/api-error.svelte';
	import { UpdateSelf } from '$lib/api/self-api';
	import { auth } from '$lib/auth.svelte';
	import { DeleteUserDialog } from '$lib/components/dialogs';
	import { Spinner } from '$lib/components';
	import { Button, Dialog, PasswordInput } from '$lib/components/ui';
	import type { SelfUpdateModel } from '$lib/models/user-model';
	import { Separator } from 'bits-ui';
	import { toast } from 'svelte-sonner';

	let isEditingDisplayName = $state(false);
	let displayNameValue = $state('');
	let editableRef = $state<HTMLDivElement>();
	let isSaving = $state(false);
	let originalValue = $state('');

	let isEditingPassword = $state(false);
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isSavingPassword = $state(false);
	let currentPasswordRef = $state<HTMLInputElement>();

	// Sync contenteditable text when not editing
	$effect(() => {
		if (!isEditingDisplayName && editableRef && auth.user?.displayName) {
			editableRef.textContent = auth.user.displayName;
			displayNameValue = auth.user.displayName;
		}
	});

	function startEditing() {
		originalValue = auth.user?.displayName || '';
		displayNameValue = originalValue;
		isEditingDisplayName = true;
		// Focus and select text after background transition starts (delay to avoid flash)
		setTimeout(() => {
			if (editableRef) {
				editableRef.textContent = displayNameValue;
				// Ensure no focus styles are applied
				editableRef.style.outline = 'none';
				editableRef.style.boxShadow = 'none';
				editableRef.focus();
				// Select all text
				const range = document.createRange();
				range.selectNodeContents(editableRef);
				const selection = window.getSelection();
				selection?.removeAllRanges();
				selection?.addRange(range);
			}
		}, 50);
	}

	function cancelEditing() {
		isEditingDisplayName = false;
		displayNameValue = originalValue;
		if (editableRef) {
			editableRef.textContent = originalValue;
		}
	}

	function handleInput() {
		if (editableRef) {
			displayNameValue = editableRef.textContent || '';
		}
	}

	async function saveDisplayName() {
		const trimmedValue = displayNameValue.trim();
		if (!trimmedValue || trimmedValue === auth.user?.displayName) {
			cancelEditing();
			return;
		}

		isSaving = true;
		try {
			await UpdateSelf({ displayName: trimmedValue } satisfies SelfUpdateModel);
			await auth.me();
			isEditingDisplayName = false;
			displayNameValue = trimmedValue;
			// Update the contenteditable element text
			if (editableRef) {
				editableRef.textContent = trimmedValue;
			}
			toast.success('Display name updated');
		} catch (error) {
			toast.error((error as APIError).message);
			// Restore original value on error
			if (editableRef) {
				editableRef.textContent = originalValue;
				displayNameValue = originalValue;
			}
		} finally {
			isSaving = false;
		}
	}

	function startEditingPassword() {
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		isEditingPassword = true;
		// Focus first input after a brief delay
		setTimeout(() => {
			currentPasswordRef?.focus();
		}, 50);
	}

	function cancelEditingPassword() {
		isEditingPassword = false;
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
	}

	async function savePassword() {
		if (newPassword !== confirmPassword) {
			toast.error('Passwords do not match');
			return;
		}

		if (!currentPassword || !newPassword || !confirmPassword) {
			toast.error('All fields are required');
			return;
		}

		isSavingPassword = true;
		try {
			await UpdateSelf({ currentPassword, password: newPassword } satisfies SelfUpdateModel);
			isEditingPassword = false;
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			toast.success('Password changed');
		} catch (error) {
			toast.error((error as APIError).message);
		} finally {
			isSavingPassword = false;
		}
	}
</script>

{#if auth.user !== null}
	<div class="container-px py-8">
		<div class="mx-auto flex max-w-2xl flex-col place-content-center items-start gap-5">
			<!-- Username -->
			<div class="flex flex-col gap-3">
				<div class="text-foreground-alt-3 text-[15px] uppercase">Username</div>
				<span class="text-background-primary text-2xl">{auth.user.username}</span>
			</div>

			<Separator.Root class="bg-background-alt-3 my-2 h-px w-full shrink-0" />

			<!-- Display name -->
			<div class="flex flex-col gap-3 w-full">
				<div class="flex flex-row items-center justify-between w-full">
					<div class="text-foreground-alt-3 text-[15px] uppercase">Display Name</div>
					{#if !isEditingDisplayName}
						<button
							type="button"
							onclick={startEditing}
							class="text-foreground-alt-3 hover:text-foreground-alt-1 cursor-pointer bg-transparent py-0 text-sm duration-200 hover:bg-transparent"
						>
							Edit
						</button>
					{:else}
						<button
							type="button"
							onclick={cancelEditing}
							class="text-foreground-alt-3 hover:text-foreground-alt-1 cursor-pointer bg-transparent py-0 text-sm duration-200 hover:bg-transparent"
						>
							Cancel
						</button>
					{/if}
				</div>
				<div class="flex flex-col gap-2">
					<div
						bind:this={editableRef}
						contenteditable={isEditingDisplayName}
						role={isEditingDisplayName ? 'textbox' : undefined}
						class="text-background-primary text-2xl block min-h-8 outline-none transition-all duration-200 {isEditingDisplayName
							? 'bg-background-alt-3 focus:bg-background-alt-4 rounded-md border-0 px-2.5 py-1.5 cursor-text focus:outline-none focus:ring-0'
							: 'cursor-default'}"
						oninput={handleInput}
						onkeydown={(e) => {
							if (e.key === 'Enter' && !e.shiftKey) {
								e.preventDefault();
								saveDisplayName();
							} else if (e.key === 'Escape') {
								e.preventDefault();
								cancelEditing();
							}
						}}
						onfocus={(e) => {
							// Prevent default focus outline and any browser focus rings
							if (e.target instanceof HTMLElement) {
								e.target.style.outline = 'none';
								e.target.style.boxShadow = 'none';
							}
						}}
					>
						{isEditingDisplayName ? displayNameValue : auth.user.displayName}
					</div>
					{#if isEditingDisplayName}
						<Button
							type="button"
							variant="default"
							onclick={saveDisplayName}
							disabled={!displayNameValue.trim() || displayNameValue === auth.user?.displayName || isSaving}
							class="w-36"
						>
							{#if isSaving}
								<Spinner class="bg-background-alt-4 size-2" />
							{:else}
								Save
							{/if}
						</Button>
					{/if}
				</div>
			</div>

			<Separator.Root class="bg-background-alt-3 my-2 h-px w-full shrink-0" />

			<!-- Role -->
			<div class="flex flex-col gap-3">
				<div class="text-foreground-alt-3 text-[15px] uppercase">Role</div>
				<span class="text-background-primary text-2xl">{auth.isAdmin ? 'Admin' : 'User'}</span>
			</div>

			<Separator.Root class="bg-background-alt-3 my-2 h-px w-full shrink-0" />

			<!-- Password -->
			<div class="flex flex-col gap-3 w-full">
				<div class="flex flex-row items-center justify-between w-full">
					<div class="text-foreground-alt-3 text-[15px] uppercase">Password</div>
					{#if !isEditingPassword}
						<button
							type="button"
							onclick={startEditingPassword}
							class="text-foreground-alt-3 hover:text-foreground-alt-1 cursor-pointer bg-transparent py-0 text-sm duration-200 hover:bg-transparent"
						>
							Edit
						</button>
					{:else}
						<button
							type="button"
							onclick={cancelEditingPassword}
							class="text-foreground-alt-3 hover:text-foreground-alt-1 cursor-pointer bg-transparent py-0 text-sm duration-200 hover:bg-transparent"
						>
							Cancel
						</button>
					{/if}
				</div>
				{#if isEditingPassword}
					<div class="flex flex-col gap-2">
						<div class="flex flex-col gap-2.5">
							<div class="text-foreground-alt-3 text-sm">Current Password:</div>
							<PasswordInput
								bind:ref={currentPasswordRef}
								bind:value={currentPassword}
								name="current password"
								onkeydown={(e) => {
									if (e.key === 'Escape') {
										e.preventDefault();
										cancelEditingPassword();
									}
								}}
							/>
						</div>
						<div class="flex flex-col gap-2.5">
							<div class="text-foreground-alt-3 text-sm">New Password:</div>
							<PasswordInput
								bind:value={newPassword}
								name="new password"
								onkeydown={(e) => {
									if (e.key === 'Enter' && !e.shiftKey) {
										e.preventDefault();
										savePassword();
									} else if (e.key === 'Escape') {
										e.preventDefault();
										cancelEditingPassword();
									}
								}}
							/>
						</div>
						<div class="flex flex-col gap-2.5">
							<div class="text-foreground-alt-3 text-sm">Confirm Password:</div>
							<PasswordInput
								bind:value={confirmPassword}
								name="confirm password"
								onkeydown={(e) => {
									if (e.key === 'Enter' && !e.shiftKey) {
										e.preventDefault();
										savePassword();
									} else if (e.key === 'Escape') {
										e.preventDefault();
										cancelEditingPassword();
									}
								}}
							/>
						</div>
						<Button
							type="button"
							variant="default"
							onclick={savePassword}
							disabled={!currentPassword || !newPassword || !confirmPassword || isSavingPassword}
							class="w-36"
						>
							{#if isSavingPassword}
								<Spinner class="bg-background-alt-4 size-2" />
							{:else}
								Save
							{/if}
						</Button>
					</div>
				{/if}
			</div>

			<Separator.Root class="bg-background-alt-3 my-2 h-px w-full shrink-0" />

			<!-- Delete account -->
			<div class="flex flex-col gap-3">
				<div class="text-foreground-alt-3 text-[15px] uppercase">Delete Account</div>
				<DeleteUserDialog value={auth.user}>
					{#snippet trigger()}
						<Dialog.Trigger
							class="bg-background-error enabled:hover:bg-background-error-alt-1 text-foreground-alt-1 enabled:hover:text-foreground w-auto"
						>
							Delete Account
						</Dialog.Trigger>
					{/snippet}
				</DeleteUserDialog>
			</div>
		</div>
	</div>
{/if}
