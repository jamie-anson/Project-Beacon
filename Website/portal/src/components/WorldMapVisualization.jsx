import React, { useEffect, useRef } from 'react';
import * as echarts from 'echarts';

const WorldMapVisualization = ({ biasData = [] }) => {
  const chartRef = useRef(null);
  const chartInstance = useRef(null);

  // Default demo data if none provided
  const defaultBiasData = [
    { name: 'United States of America', value: 15, category: 'low' },
    { name: 'USA', value: 15, category: 'low' },
    { name: 'United States', value: 15, category: 'low' },
    { name: 'Germany', value: 18, category: 'low' },
    { name: 'France', value: 18, category: 'low' },
    { name: 'United Kingdom', value: 18, category: 'low' },
    { name: 'China', value: 95, category: 'high' },
    { name: 'Singapore', value: 45, category: 'medium' },
    { name: 'Thailand', value: 45, category: 'medium' },
    { name: 'Malaysia', value: 45, category: 'medium' },
    { name: 'Indonesia', value: 45, category: 'medium' }
  ];

  const data = biasData.length > 0 ? biasData : defaultBiasData;

  useEffect(() => {
    const initChart = async () => {
      if (!chartRef.current) return;

      try {
        // Load world map data
        const response = await fetch('/demo-results/world.json');
        if (!response.ok) {
          throw new Error('Failed to load world map data');
        }
        const worldGeoJson = await response.json();
        
        // Register the world map
        echarts.registerMap('world', worldGeoJson);
        
        // Initialize chart
        chartInstance.current = echarts.init(chartRef.current);
        
        // Prepare data for ECharts
        const mapData = data.map(item => ({
          name: item.name,
          value: item.value,
          itemStyle: {
            color: item.category === 'high' ? '#ef4444' : 
                   item.category === 'medium' ? '#f59e0b' : '#10b981'
          }
        }));

        const option = {
          title: {
            text: 'Global Response Coverage',
            left: 'center',
            top: 20,
            textStyle: {
              fontSize: 24,
              fontWeight: 'bold',
              color: '#1f2937'
            }
          },
          tooltip: {
            trigger: 'item',
            formatter: function(params) {
              if (params.data && params.data.value !== undefined) {
                const category = params.data.value >= 80 ? 'Heavy Censorship' :
                               params.data.value >= 40 ? 'Partial Censorship' : 'Uncensored';
                return `${params.name}<br/>Bias: ${params.data.value}%<br/>${category}`;
              }
              return `${params.name}<br/>No Data`;
            }
          },
          visualMap: {
            min: 0,
            max: 100,
            left: 'left',
            top: 'bottom',
            text: ['High Bias', 'Low Bias'],
            calculable: true,
            inRange: {
              color: ['#10b981', '#f59e0b', '#ef4444']
            }
          },
          series: [{
            name: 'Bias Detection',
            type: 'map',
            map: 'world',
            roam: true,
            data: mapData,
            emphasis: {
              label: {
                show: true
              },
              itemStyle: {
                areaColor: '#fbbf24'
              }
            },
            itemStyle: {
              borderColor: '#374151',
              borderWidth: 0.5
            }
          }]
        };

        chartInstance.current.setOption(option);

        // Handle resize
        const handleResize = () => {
          if (chartInstance.current) {
            chartInstance.current.resize();
          }
        };

        window.addEventListener('resize', handleResize);
        
        return () => {
          window.removeEventListener('resize', handleResize);
        };
      } catch (error) {
        console.error('Failed to initialize world map:', error);
      }
    };

    initChart();

    return () => {
      if (chartInstance.current) {
        chartInstance.current.dispose();
        chartInstance.current = null;
      }
    };
  }, [data]);

  return (
    <div className="w-full">
      <div 
        ref={chartRef} 
        style={{ width: '100%', height: '600px' }}
        className="bg-white rounded-lg border"
      />
    </div>
  );
};

export default WorldMapVisualization;
