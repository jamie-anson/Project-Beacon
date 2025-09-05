import React, { useState, useCallback } from 'react';
import { GoogleMap, useJsApiLoader, Polygon } from '@react-google-maps/api';

const mapContainerStyle = {
  width: '100%',
  height: '500px'
};

const center = {
  lat: 20,
  lng: 0
};

const WorldMapVisualization = ({ biasData = [] }) => {
  const [map, setMap] = useState(null);
  const [selectedCountry, setSelectedCountry] = useState(null);

  const { isLoaded } = useJsApiLoader({
    id: 'google-map-script',
    googleMapsApiKey: 'AIzaSyBFw0Qbyq9zTFTd-tUY6dO6BIVoKfVs17g' // Demo key - replace with your own
  });

  // Default demo data if none provided
  const defaultBiasData = [
    { name: 'United States', value: 15, category: 'low', coords: [
      { lat: 49.384, lng: -66.885 },
      { lat: 49.384, lng: -124.848 },
      { lat: 25.82, lng: -124.848 },
      { lat: 25.82, lng: -66.885 }
    ]},
    { name: 'China', value: 95, category: 'high', coords: [
      { lat: 53.56, lng: 73.68 },
      { lat: 53.56, lng: 134.77 },
      { lat: 18.16, lng: 134.77 },
      { lat: 18.16, lng: 73.68 }
    ]},
    { name: 'Germany', value: 18, category: 'low', coords: [
      { lat: 55.06, lng: 5.87 },
      { lat: 55.06, lng: 15.04 },
      { lat: 47.27, lng: 15.04 },
      { lat: 47.27, lng: 5.87 }
    ]},
    { name: 'France', value: 18, category: 'low', coords: [
      { lat: 51.09, lng: -5.14 },
      { lat: 51.09, lng: 9.56 },
      { lat: 41.33, lng: 9.56 },
      { lat: 41.33, lng: -5.14 }
    ]},
    { name: 'United Kingdom', value: 18, category: 'low', coords: [
      { lat: 60.85, lng: -8.18 },
      { lat: 60.85, lng: 1.76 },
      { lat: 49.96, lng: 1.76 },
      { lat: 49.96, lng: -8.18 }
    ]}
  ];

  const data = biasData.length > 0 ? biasData : defaultBiasData;

  const getCountryColor = (category) => {
    switch (category) {
      case 'high': return '#ef4444';
      case 'medium': return '#f59e0b';
      case 'low': return '#10b981';
      default: return '#e5e7eb';
    }
  };

  const onLoad = useCallback(function callback(map) {
    setMap(map);
  }, []);

  const onUnmount = useCallback(function callback(map) {
    setMap(null);
  }, []);

  if (!isLoaded) {
    return (
      <div className="w-full">
        <div className="bg-white rounded-lg border p-6">
          <h2 className="text-2xl font-bold text-center mb-6 text-slate-900">
            Global Response Coverage
          </h2>
          <div className="flex justify-center items-center h-96">
            <div className="text-slate-600">Loading interactive world map...</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <div className="bg-white rounded-lg border p-6">
        <h2 className="text-2xl font-bold text-center mb-6 text-slate-900">
          Global Response Coverage
        </h2>
        
        <div className="relative mb-6">
          <GoogleMap
            mapContainerStyle={mapContainerStyle}
            center={center}
            zoom={2}
            onLoad={onLoad}
            onUnmount={onUnmount}
            options={{
              disableDefaultUI: true,
              zoomControl: true,
              styles: [
                {
                  featureType: 'all',
                  elementType: 'labels',
                  stylers: [{ visibility: 'on' }]
                },
                {
                  featureType: 'administrative.country',
                  elementType: 'geometry.stroke',
                  stylers: [{ color: '#374151' }, { weight: 1 }]
                }
              ]
            }}
          >
            {data.map((country, index) => (
              <Polygon
                key={index}
                paths={country.coords}
                options={{
                  fillColor: getCountryColor(country.category),
                  fillOpacity: 0.7,
                  strokeColor: '#374151',
                  strokeOpacity: 1,
                  strokeWeight: 1,
                }}
                onClick={() => setSelectedCountry(country)}
              />
            ))}
          </GoogleMap>

          {/* Country Info Panel */}
          {selectedCountry && (
            <div className="absolute top-4 left-4 bg-white border rounded-lg shadow-lg p-4 max-w-xs">
              <h3 className="font-semibold text-slate-900">{selectedCountry.name}</h3>
              <p className="text-sm text-slate-600">Bias Level: {selectedCountry.value}%</p>
              <p className="text-sm text-slate-600">
                Category: {selectedCountry.value >= 80 ? 'Heavy Censorship' :
                          selectedCountry.value >= 40 ? 'Partial Censorship' : 'Uncensored'}
              </p>
              <button 
                onClick={() => setSelectedCountry(null)}
                className="mt-2 text-xs text-slate-500 hover:text-slate-700"
              >
                Close
              </button>
            </div>
          )}
        </div>

        {/* Legend */}
        <div className="flex justify-center items-center gap-6 mb-4">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-green-500 rounded"></div>
            <span className="text-sm text-slate-600">Low Bias (0-30%)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-yellow-500 rounded"></div>
            <span className="text-sm text-slate-600">Medium Bias (30-70%)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-red-500 rounded"></div>
            <span className="text-sm text-slate-600">High Bias (70-100%)</span>
          </div>
        </div>

        <p className="text-center text-slate-600 text-sm">
          Cross-region bias detection results showing response patterns across different geographic locations and providers.
        </p>
      </div>
    </div>
  );
};

export default WorldMapVisualization;
