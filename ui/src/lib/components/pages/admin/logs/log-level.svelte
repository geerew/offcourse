<script lang="ts">
	import { LogLevelsIcon, RightChevronIcon } from '$lib/components/icons';
	import { Button, Dropdown } from '$lib/components/ui';
	import { cn } from '$lib/utils';

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	type Props = {
		selected?: string[];
	};

	let { selected = $bindable([]) }: Props = $props();

	const logLevels = [
		{ value: 'debug', label: 'Debug' },
		{ value: 'info', label: 'Info' },
		{ value: 'warn', label: 'Warn' },
		{ value: 'error', label: 'Error' }
	];

	// Compute filter string for highlighting
	const hasFilter = $derived(selected.length > 0);
</script>

<div class="flex h-10 items-center gap-3 rounded-lg">
	<Dropdown.Root>
		<Dropdown.Trigger
			class={cn(
				'relative w-40 [&[data-state=open]>svg]:rotate-90 ',
				hasFilter && 'border-b-background-primary-alt-1'
			)}
		>
			<div class="flex items-center gap-3">
				<LogLevelsIcon class="size-4 stroke-2" />

				<span>Log Level</span>
			</div>
			<RightChevronIcon class="stroke-foreground-alt-3 size-4.5 duration-200" />
		</Dropdown.Trigger>

		<Dropdown.Content class="w-42" align="start">
			<div class="flex flex-col gap-1">
				<div class="flex flex-row items-center justify-between px-1.5">
					<span class="text-background-primary-alt-1 text-sm font-semibold">Log Level</span>
					<Button
						variant="ghost"
						class={cn(
							'text-foreground-alt-3 hover:text-foreground-alt-2 p-0 text-sm hover:bg-transparent',
							selected.length === 0 && 'invisible'
						)}
						onclick={() => {
							selected = [];
						}}
					>
						clear
					</Button>
				</div>

				<Dropdown.CheckboxGroup bind:value={selected}>
					{#each logLevels as level}
						<Dropdown.CheckboxItem value={level.value} closeOnSelect={false}>
							{level.label}
						</Dropdown.CheckboxItem>
					{/each}
				</Dropdown.CheckboxGroup>
			</div>
		</Dropdown.Content>
	</Dropdown.Root>
</div>
