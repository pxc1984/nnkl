<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import type cytoscape from "cytoscape";
  import type { GraphData, GraphNode } from "$lib/data/graph";
  import { NODE_COLORS } from "$lib/data/graph";

  interface Props {
    data: GraphData;
    onNodeSelect?: (node: GraphNode | null) => void;
  }

  let { data, onNodeSelect }: Props = $props();

  let container: HTMLDivElement | null = $state(null);
  let cy: cytoscape.Core | null = $state(null);

  onMount(async () => {
    if (!container) return;

    const cytoscapeLib = (await import("cytoscape")).default;

    cy = cytoscapeLib({
      container,
      elements: [
        ...data.nodes.map((n) => ({
          data: {
            id: n.id,
            label: n.label,
            type: n.type,
            color: NODE_COLORS[n.type] ?? NODE_COLORS.Unknown,
          },
        })),
        ...data.edges.map((e, index) => ({
          data: {
            id: `edge-${index}`,
            source: e.source,
            target: e.target,
            label: e.label,
          },
        })),
      ],
      style: [
        {
          selector: "node",
          style: {
            "background-color": "data(color)",
            label: "data(label)",
            width: 40,
            height: 40,
            "font-size": "12px",
            "text-valign": "bottom",
            "text-halign": "center",
            color: "#ffffff",
            "text-outline-color": "#000000",
            "text-outline-width": 1,
            "text-background-color": "#000000",
            "text-background-opacity": 0.6,
            "text-background-padding": "2px",
            "text-background-shape": "roundrectangle",
          },
        },
        {
          selector: "edge",
          style: {
            width: 2,
            "line-color": "#94a3b8",
            "target-arrow-color": "#94a3b8",
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            label: "data(label)",
            "font-size": "10px",
            color: "#e2e8f0",
            "text-outline-color": "#000000",
            "text-outline-width": 1,
          },
        },
        {
          selector: ":selected",
          style: {
            "border-width": 4,
            "border-color": "#facc15",
          },
        },
      ],
      layout: {
        name: "cose",
        padding: 20,
        animate: true,
        animationDuration: 500,
        fit: true,
      },
      wheelSensitivity: 0.2,
      minZoom: 0.2,
      maxZoom: 3,
    });

    cy.on("tap", "node", (event) => {
      const node = event.target;
      const nodeData = data.nodes.find((n) => n.id === node.id()) ?? null;
      onNodeSelect?.(nodeData);
    });

    cy.on("tap", (event) => {
      if (event.target === cy) {
        onNodeSelect?.(null);
      }
    });
  });

  onDestroy(() => {
    cy?.destroy();
  });
</script>

<div bind:this={container} class="h-full w-full bg-background"></div>
