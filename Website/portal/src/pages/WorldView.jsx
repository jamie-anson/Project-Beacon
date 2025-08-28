import React, { useEffect, useMemo, useRef, useState } from 'react';
import { getGeo } from '../lib/api.js';
import { useQuery } from '../state/useQuery.js';

// Minimal ISO2 -> ECharts world map name mapping for countries we synthesize
const ISO2_TO_NAME = {
  US: 'United States',
  CN: 'China',
  FR: 'France',
  GB: 'United Kingdom',
  DE: 'Germany',
  IN: 'India',
  JP: 'Japan',
  TW: 'Taiwan',
  HK: 'Hong Kong',
  RU: 'Russia',
  BR: 'Brazil',
  CA: 'Canada',
  AU: 'Australia',
  ZA: 'South Africa',
  NG: 'Nigeria',
  MX: 'Mexico',
  ES: 'Spain',
  IT: 'Italy',
  KR: 'South Korea',
  SG: 'Singapore',
};

export default function WorldView() {
  const chartRef = useRef(null);
  const [echarts, setEcharts] = useState(null);
  const [worldGeo, setWorldGeo] = useState(null);

  const { data, loading, error, refetch } = useQuery('geo:countries', getGeo, { interval: 30000 });
  const seriesData = useMemo(() => {
    const countries = data?.countries || {};
    return Object.entries(countries).map(([code, value]) => ({
      name: ISO2_TO_NAME[code] || code,
      value,
    }));
  }, [data]);

  useEffect(() => {
    let mounted = true;
    // Lazy-load echarts (modular) and world geojson
    (async () => {
      try {
        const [core, charts, comps, renderer] = await Promise.all([
          import('echarts/core'),
          import('echarts/charts'), // MapChart
          import('echarts/components'), // TooltipComponent, VisualMapComponent
          import('echarts/renderers'), // CanvasRenderer
        ]);
        if (!mounted) return;
        const ech = core;
        const { MapChart } = charts;
        const { TooltipComponent, VisualMapComponent } = comps;
        const { CanvasRenderer } = renderer;
        // Register required pieces
        (ech.default || ech).use([MapChart, TooltipComponent, VisualMapComponent, CanvasRenderer]);
        setEcharts(ech.default || ech);

        // Zero-network local topojson import (preferred)
        let geo = null;
        try {
          const [{ feature }, world] = await Promise.all([
            import('topojson-client'),
            import('world-atlas/countries-110m.json'),
          ]);
          const topo = (world && (world.default || world));
          if (topo && topo.objects && topo.objects.countries) {
            geo = feature(topo, topo.objects.countries);
          }
        } catch {}
        // External fallbacks if local import fails
        if (!geo) {
          const sources = [
            'https://raw.githubusercontent.com/apache/echarts/5.5.0/map/json/world.json',
            'https://raw.githubusercontent.com/apache/echarts/5.4.3/map/json/world.json',
            'https://fastly.jsdelivr.net/npm/echarts@5.5.0/map/json/world.json',
            'https://cdn.jsdelivr.net/npm/echarts@5.5.0/map/json/world.json',
          ];
          for (const url of sources) {
            try {
              const res = await fetch(url, { cache: 'force-cache' });
              if (res.ok) { geo = await res.json(); break; }
            } catch {}
          }
        }
        if (!geo) throw new Error('Failed to load world geojson from all sources');
        if (!mounted) return;
        setWorldGeo(geo);
      } catch (e) {
        console.warn('[WorldView] Failed modular load, trying full build:', e);
        try {
          const ech = await import('echarts');
          if (!mounted) return;
          setEcharts(ech.default || ech);
          const res = await fetch('https://unpkg.com/echarts@5/map/json/world.json');
          const geo = await res.json();
          if (!mounted) return;
          setWorldGeo(geo);
        } catch (e2) {
          console.error('[WorldView] Full build fallback failed:', e2);
        }
      }
    })();
    return () => { mounted = false; };
  }, []);

  useEffect(() => {
    if (!echarts || !worldGeo || !chartRef.current) return;
    const chart = echarts.init(chartRef.current);
    try { chart.showLoading('default', { text: 'Loading world map…' }); } catch {}
    echarts.registerMap('world', worldGeo);

    const maxVal = seriesData.reduce((m, x) => Math.max(m, x.value || 0), 0) || 1;

    chart.setOption({
      tooltip: {
        trigger: 'item',
        formatter: (p) => `${p.name}: ${p.value ?? 0}`,
      },
      visualMap: {
        min: 0,
        max: maxVal,
        left: 'left',
        bottom: 10,
        inRange: { color: ['#e2f3ff', '#66a3ff', '#004cce'] },
        text: ['High', 'Low'],
        calculable: true,
      },
      series: [
        {
          name: 'Responses',
          type: 'map',
          map: 'world',
          roam: true,
          emphasis: { label: { show: false } },
          data: seriesData,
        },
      ],
    });
    try { chart.hideLoading(); } catch {}

    const handle = () => chart.resize();
    window.addEventListener('resize', handle);
    // Ensure first paint size
    setTimeout(() => { try { chart.resize(); } catch {} }, 0);
    return () => {
      window.removeEventListener('resize', handle);
      chart.dispose();
    };
  }, [echarts, worldGeo, seriesData]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">World View</h2>
        <div className="flex items-center gap-2 text-sm">
          <button onClick={refetch} className="px-2 py-1 border rounded hover:bg-slate-50">Refresh</button>
        </div>
      </div>

      {/* Intro: why provider country of origin matters */}
      <div className="bg-white border rounded p-4 text-sm text-slate-700">
        <p className="mb-2">
          The country of origin of an AI provider often correlates with the data sources, labeling norms, and
          policy constraints used to train and align their models. These differences can shape what models
          consider correct, safe, or culturally appropriate.
        </p>
        <ul className="list-disc pl-5 space-y-1">
          <li><strong>Data & culture:</strong> Local language coverage and cultural priors can shift answers and tone.</li>
          <li><strong>Regulation & safety:</strong> National policies influence what content is filtered or prioritized.</li>
          <li><strong>Market focus:</strong> Providers optimize for their primary user markets, affecting examples and assumptions.</li>
        </ul>
        <p className="mt-2">
          This map summarizes where responses are attributed across countries so you can spot geographic patterns
          in performance or behavior. Counts are synthetic for demo purposes and do not represent real user data.
        </p>
      </div>

      {loading && <div className="text-sm text-slate-500">Loading geo data…</div>}
      {error && <div className="text-sm text-red-600">Failed to load geo data.</div>}

      <div className="bg-white border rounded h-[560px]">
        <div ref={chartRef} className="w-full h-full" />
      </div>

      <p className="text-xs text-slate-500">
        Synthetic geo distribution for demo purposes. Replace with real origin metadata when available.
      </p>
    </div>
  );
}
