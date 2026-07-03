<script lang="ts">
    import LoginForm from "$lib/components/login-form.svelte";
    import Logo from "$lib/components/Logo.svelte";
</script>
<div class="grid min-h-svh lg:grid-cols-2">
    <div class="flex flex-col gap-4 p-6 md:p-10">
        <Logo/>
        <div class="flex flex-1 items-center justify-center">
            <div class="w-full max-w-xs">
                <LoginForm/>
            </div>
        </div>
    </div>
    <div class="bg-muted relative hidden lg:block">
        <img
                src="https://nornickel.ru/images/main-page/owl.jpg"
                alt="placeholder"
                class="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale"
        />
    </div>
</div>