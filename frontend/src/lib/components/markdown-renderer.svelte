<script lang="ts">
	import { marked, type Tokens } from "marked";

	let { markdown }: { markdown: string } = $props();

	type MarkdownToken = Tokens.Generic;

	const tokens = $derived(marked.lexer(markdown ?? ""));

	function inlineTokens(text: string): MarkdownToken[] {
		return marked.Lexer.lexInline(text ?? "");
	}

	function hrefLabel(token: MarkdownToken): string {
		return "text" in token && typeof token.text === "string" ? token.text : token.raw;
	}
</script>

{#snippet renderInline(items: MarkdownToken[])}
	{#each items as item, index (`${item.type}-${index}`)}
		{#if item.type === "text"}
			{item.raw}
		{:else if item.type === "strong"}
			<strong>{@render renderInline(item.tokens ?? inlineTokens(item.raw))}</strong>
		{:else if item.type === "em"}
			<em>{@render renderInline(item.tokens ?? inlineTokens(item.raw))}</em>
		{:else if item.type === "codespan"}
			<code>{item.text}</code>
		{:else if item.type === "br"}
			<br />
		{:else if item.type === "link"}
			<span class="text-primary underline underline-offset-4">
				{@render renderInline(item.tokens ?? inlineTokens(hrefLabel(item)))}
			</span>
		{:else if item.type === "del"}
			<s>{@render renderInline(item.tokens ?? inlineTokens(item.raw))}</s>
		{:else}
			{item.raw}
		{/if}
	{/each}
{/snippet}

{#snippet renderBlocks(items: MarkdownToken[])}
	{#each items as item, index (`${item.type}-${index}`)}
		{#if item.type === "heading"}
			{#if item.depth === 1}
				<h1>{@render renderInline(item.tokens ?? [])}</h1>
			{:else if item.depth === 2}
				<h2>{@render renderInline(item.tokens ?? [])}</h2>
			{:else if item.depth === 3}
				<h3>{@render renderInline(item.tokens ?? [])}</h3>
			{:else}
				<h4>{@render renderInline(item.tokens ?? [])}</h4>
			{/if}
		{:else if item.type === "paragraph"}
			<p>{@render renderInline(item.tokens ?? [])}</p>
		{:else if item.type === "list"}
			{#if item.ordered}
				<ol>
					{#each item.items as listItem, listIndex (`${listItem.text}-${listIndex}`)}
						<li>{@render renderInline(listItem.tokens ?? inlineTokens(listItem.text))}</li>
					{/each}
				</ol>
			{:else}
				<ul>
					{#each item.items as listItem, listIndex (`${listItem.text}-${listIndex}`)}
						<li>{@render renderInline(listItem.tokens ?? inlineTokens(listItem.text))}</li>
					{/each}
				</ul>
			{/if}
		{:else if item.type === "blockquote"}
			<blockquote>
				{@render renderBlocks(item.tokens ?? [])}
			</blockquote>
		{:else if item.type === "code"}
			<pre><code>{item.text}</code></pre>
		{:else if item.type === "hr"}
			<hr />
		{:else if item.type !== "space"}
			<p>{item.raw}</p>
		{/if}
	{/each}
{/snippet}

<div class="prose prose-invert prose-pre:bg-background prose-pre:border prose-pre:border-border/60 prose-code:text-foreground max-w-none">
	{@render renderBlocks(tokens)}
</div>
