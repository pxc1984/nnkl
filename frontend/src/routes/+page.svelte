<script lang="ts">
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { authState } from "$lib/auth/store";

	$effect(() => {
		if (!browser) {
			return;
		}

		const unsubscribe = authState.subscribe((state) => {
			if (!state.ready) {
				return;
			}

			void goto(state.user ? resolve("/data/ask") : resolve("/auth/login"));
		});

		return unsubscribe;
	});
</script>