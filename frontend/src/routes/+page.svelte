<script lang="ts">
  import { onMount } from 'svelte'

  import { API_URL } from '$lib/config'
  import { Badge } from '$lib/components/ui/badge'
  import * as Card from '$lib/components/ui/card'
  import { getHealth } from '$lib/api/health'
  import { Activity } from '@lucide/svelte';

  let healthStatus = 'Loading...'
  let healthDetails = `GET ${API_URL || '(same origin)'}/api/health`

  onMount(async () => {
    try {
      const health = await getHealth()
      healthStatus = typeof health === 'string' ? health : JSON.stringify(health)
    } catch (error) {
      healthStatus = 'Request failed'
      healthDetails = error instanceof Error ? error.message : 'Unknown error'
    }
  })
</script>

<main class="min-h-screen bg-background text-foreground">
  <div class="mx-auto flex min-h-screen w-full max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6 lg:px-8">
    <section class="grid gap-4 md:grid-cols-3">
      <Card.Root>
        <Card.Header>
          <Card.Action>
            <Activity class="size-4 text-muted-foreground" />
          </Card.Action>
          <Card.Title>/api/health проверка</Card.Title>
          <Card.Description>Вот что вернул бек на /api/health</Card.Description>
        </Card.Header>
        <Card.Content class="space-y-2">
          <p class="text-sm text-muted-foreground">{healthDetails}</p>
          <Badge variant="outline" class="max-w-full truncate">{healthStatus}</Badge>
        </Card.Content>
      </Card.Root>
    </section>
  </div>
</main>
