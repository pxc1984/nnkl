<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Field from "$lib/components/ui/field/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import type { HTMLAttributes } from "svelte/elements";
	import {resolve} from "$app/paths";

	let { class: className, ...restProps }: HTMLAttributes<HTMLFormElement> = $props();
</script>

<form class={cn("flex flex-col gap-6", className)} {...restProps}>
	<Field.Group>
		<div class="flex flex-col items-center gap-1 text-center">
			<h1 class="text-2xl font-bold">Создайте ваш аккаунт</h1>
		</div>
		<Field.Field>
			<Field.Label for="name">Полное имя</Field.Label>
			<Input id="name" type="text" placeholder="Владимир Потанин" required />
		</Field.Field>
		<Field.Field>
			<Field.Label for="email">Почта</Field.Label>
			<Input id="email" type="email" placeholder="potanin@nornickel.ru" required />
		</Field.Field>
		<Field.Field>
			<Field.Label for="password">Пароль</Field.Label>
			<Input id="password" type="password" required />
			<Field.Description>Должен быть минимум 8 символов.</Field.Description>
		</Field.Field>
		<Field.Field>
			<Field.Label for="confirm-password">Подтверждение пароля</Field.Label>
			<Input id="confirm-password" type="password" required />
			<Field.Description>Пожалуйста, подтвердите ваш пароль.</Field.Description>
		</Field.Field>
		<Field.Field>
			<Button type="submit">Создать аккаунт</Button>
		</Field.Field>
		<Field.Separator>Или продолжить через</Field.Separator>
		<Field.Field>
			<Field.Description class="px-6 text-center">
				Уже есть аккаунт? <a href={resolve('/auth/login')}>Войти</a>
			</Field.Description>
		</Field.Field>
	</Field.Group>
</form>
