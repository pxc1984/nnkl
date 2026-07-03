<script lang="ts">
	import {
		FieldGroup,
		Field,
		FieldLabel,
		FieldDescription,
		FieldSeparator,
	} from "$lib/components/ui/field/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLFormAttributes } from "svelte/elements";
	import {resolve} from "$app/paths";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithElementRef<HTMLFormAttributes> = $props();

	const id = $props.id();
</script>

<form class={cn("flex flex-col gap-6", className)} bind:this={ref} {...restProps}>
	<FieldGroup>
		<div class="flex flex-col items-center gap-1 text-center">
			<h1 class="text-2xl font-bold">Войти в ваш аккаунт</h1>
		</div>
		<Field>
			<FieldLabel for="email-{id}">Почта</FieldLabel>
			<Input id="email-{id}" type="email" placeholder="potanin@nornickel.ru" required />
		</Field>
		<Field>
			<div class="flex items-center">
				<FieldLabel for="password-{id}">Пароль</FieldLabel>
			</div>
			<Input id="password-{id}" type="password" required />
		</Field>
		<Field>
			<Button type="submit">Продолжить</Button>
		</Field>
		<FieldSeparator>Или можете продолжить через</FieldSeparator>
		<Field>
			<FieldDescription class="text-center">
				Нет аккаунта?
				<a href={resolve('/signup')} class="underline underline-offset-4">Зарегистрироваться</a>
			</FieldDescription>
		</Field>
	</FieldGroup>
</form>
