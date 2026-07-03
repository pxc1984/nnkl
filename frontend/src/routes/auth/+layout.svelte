<script>
    import Logo from "$lib/components/Logo.svelte";
    import RandomMainPageImage from "$lib/components/RandomMainPageImage.svelte";
</script>

<div class="grid min-h-svh lg:grid-cols-2">
    <div class="flex flex-col gap-4 p-6 md:p-10">
        <Logo/>
        <div class="flex flex-1 items-center justify-center">
            <div class="w-full max-w-xs">
                <slot />
            </div>
        </div>
    </div>
    <div class="bg-muted relative hidden lg:block">
        <RandomMainPageImage />
    </div>
</div>
