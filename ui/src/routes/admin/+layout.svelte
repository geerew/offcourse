<script lang="ts">
	import { page } from '$app/state';
	import {
		BurgerMenuIcon,
		CourseIcon,
		LogsIcon,
		ScanIcon,
		TagIcon,
		UserIcon
	} from '$lib/components/icons';
	import AdminFooter from '$lib/components/admin-footer.svelte';
	import { Badge, Button } from '$lib/components/ui';
	import { scanStore } from '$lib/scanStore.svelte';
	import { cn, remCalc } from '$lib/utils';
	import { Dialog } from 'bits-ui';
	import { innerWidth } from 'svelte/reactivity/window';
	import theme from 'tailwindcss/defaultTheme';

	let { children } = $props();

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	let menuPopupMode = $state(false);
	let dialogOpen = $state(false);

	let windowWidth = $derived(remCalc(innerWidth.current ?? 0));

	// Use scanStore for scan count
	let scanCount = $derived(scanStore.scanCount);

	const menu = [
		{
			label: 'Courses',
			href: '/admin/courses',
			matcher: '/admin/courses/',
			icon: CourseIcon
		},
		{
			label: 'Scans',
			href: '/admin/scans',
			matcher: '/admin/scans/',
			icon: ScanIcon
		},
		{
			label: 'Tags',
			href: '/admin/tags',
			matcher: '/admin/tags/',
			icon: TagIcon
		},
		{
			label: 'Users',
			href: '/admin/users',
			matcher: '/admin/users/',
			icon: UserIcon
		},
		{
			label: 'Logs',
			href: '/admin/logs',
			matcher: '/admin/logs/',
			icon: LogsIcon
		}
	];

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Register with scanStore
	$effect(() => {
		return scanStore.register();
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Set the menu popup mode based on the screen size
	$effect(() => {
		menuPopupMode = windowWidth >= +theme.screens.lg.replace('rem', '') ? false : true;
	});
</script>

{#snippet menuContents(mobile: boolean)}
	{#each menu as item}
		<Button
			href={item.href}
			variant="ghost"
			class={cn(
				'text-foreground-alt-2 hover:text-foreground hover:bg-background-alt-1 relative h-auto justify-start gap-3 px-2.5 leading-6',
				page.url.pathname.startsWith(item.matcher) &&
					'bg-background-alt-1 after:bg-background-primary after:absolute after:right-0 after:top-0 after:h-full after:w-1',
				mobile ? 'py-6 text-base' : 'py-3'
			)}
			onclick={() => {
				if (mobile && menuPopupMode) {
					dialogOpen = false;
				}
			}}
			aria-current={page.url.pathname === item.matcher}
		>
			<item.icon class="size-6 stroke-[1.5]" />
			<span>{item.label}</span>
			{#if item.label === 'Scans' && scanCount > 0}
				<Badge class="bg-background-alt-4 text-foreground ml-auto mr-2.5 text-xs">
					{scanCount}
				</Badge>
			{/if}
		</Button>
	{/each}
{/snippet}

<div
	class="flex min-h-[calc(100dvh-(var(--header-height)+1px))] flex-col pt-[calc(var(--header-height)+1)]"
>
	{#if menuPopupMode}
		<!-- Mobile layout: flex column -->
		<div class="flex flex-1 flex-col">
			<!-- Popup trigger -->
			<div class="border-background-alt-3 flex h-12 shrink-0 border-b">
				<div class="container-pl flex h-full items-center">
					<Button
						variant="ghost"
						class="text-foreground-alt-2 hover:text-foreground h-auto hover:bg-transparent"
						onclick={() => {
							dialogOpen = !dialogOpen;
						}}
					>
						<BurgerMenuIcon class="size-6 stroke-[1.5]" />
						<span>Menu</span>
					</Button>
				</div>
			</div>

			<main class="container-px flex w-full flex-1 pb-8 pt-8">
				{@render children()}
			</main>
		</div>

		<!-- Mobile menu dialog -->
		<Dialog.Root bind:open={dialogOpen}>
			<Dialog.Portal>
				<Dialog.Overlay
					class="data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/30"
				/>

				<Dialog.Content
					class="border-foreground-alt-4 bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left w-70 fixed left-0 top-0 z-50 h-full border-r pl-4 pt-4"
				>
					<nav class="flex h-full w-full flex-col gap-3 overflow-y-auto overflow-x-hidden">
						<div class="flex flex-1 flex-col gap-3">
							{@render menuContents(true)}
						</div>
						<AdminFooter />
					</nav>
				</Dialog.Content>
			</Dialog.Portal>
		</Dialog.Root>
	{:else}
		<!-- Desktop layout: grid with sidebar -->
		<div class="grid flex-1 grid-cols-[var(--settings-menu-width)_1fr] grid-rows-1 gap-6">
			<div class="relative row-span-full">
				<div class="absolute inset-0">
					<nav
						class="container-pl border-foreground-alt-5 sticky left-0 top-[calc(var(--header-height)+1px)] flex h-[calc(100dvh-(var(--header-height)+1px))] w-[--settings-menu-width] flex-col gap-4 border-r pt-8"
					>
						<div class="flex flex-1 flex-col gap-4">
							{@render menuContents(false)}
						</div>
						<AdminFooter />
					</nav>
				</div>
			</div>

			<main class="container-px flex w-full pb-8 pt-8">
				{@render children()}
			</main>
		</div>
	{/if}
</div>
