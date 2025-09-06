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
    googleMapsApiKey: import.meta.env.VITE_GOOGLE_MAPS_API_KEY
  });

  // Realistic country boundary coordinates (simplified GeoJSON-based)
  const defaultBiasData = [
    { 
      name: 'United States', 
      value: 15, 
      category: 'low', 
      coords: [
        { lat: 49.38, lng: -66.96 }, { lat: 44.83, lng: -66.96 }, { lat: 44.11, lng: -68.03 },
        { lat: 43.98, lng: -69.06 }, { lat: 41.80, lng: -69.89 }, { lat: 40.17, lng: -74.26 },
        { lat: 38.46, lng: -74.90 }, { lat: 35.00, lng: -75.41 }, { lat: 32.03, lng: -80.86 },
        { lat: 28.85, lng: -80.03 }, { lat: 25.20, lng: -80.31 }, { lat: 25.20, lng: -81.17 },
        { lat: 25.20, lng: -97.13 }, { lat: 26.00, lng: -97.13 }, { lat: 28.85, lng: -103.94 },
        { lat: 31.33, lng: -106.63 }, { lat: 31.78, lng: -108.24 }, { lat: 32.54, lng: -114.81 },
        { lat: 32.72, lng: -117.12 }, { lat: 34.45, lng: -120.64 }, { lat: 38.82, lng: -123.23 },
        { lat: 40.58, lng: -124.21 }, { lat: 43.58, lng: -124.39 }, { lat: 46.23, lng: -124.56 },
        { lat: 48.38, lng: -124.79 }, { lat: 48.99, lng: -123.43 }, { lat: 48.99, lng: -95.15 },
        { lat: 49.38, lng: -95.15 }
      ]
    },
    { 
      name: 'China', 
      value: 95, 
      category: 'high', 
      coords: [
        { lat: 53.56, lng: 109.78 }, { lat: 53.26, lng: 119.28 }, { lat: 50.33, lng: 127.28 },
        { lat: 48.86, lng: 133.68 }, { lat: 42.55, lng: 130.63 }, { lat: 40.09, lng: 124.27 },
        { lat: 35.60, lng: 119.30 }, { lat: 31.89, lng: 120.22 }, { lat: 22.30, lng: 114.18 },
        { lat: 20.92, lng: 110.15 }, { lat: 21.48, lng: 108.05 }, { lat: 23.88, lng: 97.40 },
        { lat: 28.34, lng: 89.47 }, { lat: 32.36, lng: 78.73 }, { lat: 35.49, lng: 74.98 },
        { lat: 40.05, lng: 73.67 }, { lat: 42.52, lng: 80.11 }, { lat: 44.29, lng: 87.36 },
        { lat: 47.75, lng: 90.28 }, { lat: 50.33, lng: 106.34 }
      ]
    },
    { 
      name: 'Germany', 
      value: 18, 
      category: 'low', 
      coords: [
        { lat: 55.06, lng: 8.41 }, { lat: 54.98, lng: 10.79 }, { lat: 54.36, lng: 13.80 },
        { lat: 54.08, lng: 14.12 }, { lat: 53.56, lng: 14.12 }, { lat: 52.05, lng: 14.89 },
        { lat: 50.92, lng: 14.62 }, { lat: 50.42, lng: 12.24 }, { lat: 49.97, lng: 12.24 },
        { lat: 49.48, lng: 11.42 }, { lat: 47.53, lng: 10.43 }, { lat: 47.27, lng: 9.59 },
        { lat: 47.56, lng: 8.32 }, { lat: 47.69, lng: 7.59 }, { lat: 49.01, lng: 8.16 },
        { lat: 50.11, lng: 6.24 }, { lat: 51.48, lng: 5.99 }, { lat: 53.63, lng: 7.09 }
      ]
    },
    { 
      name: 'France', 
      value: 18, 
      category: 'low', 
      coords: [
        { lat: 51.09, lng: 2.54 }, { lat: 50.95, lng: 1.34 }, { lat: 50.13, lng: -1.27 },
        { lat: 48.68, lng: -1.68 }, { lat: 48.40, lng: -4.78 }, { lat: 47.28, lng: -2.96 },
        { lat: 43.49, lng: -1.90 }, { lat: 42.56, lng: 1.73 }, { lat: 42.51, lng: 3.03 },
        { lat: 43.37, lng: 7.41 }, { lat: 43.75, lng: 7.41 }, { lat: 44.35, lng: 6.53 },
        { lat: 45.13, lng: 6.80 }, { lat: 45.78, lng: 6.05 }, { lat: 46.46, lng: 6.84 },
        { lat: 47.55, lng: 7.59 }, { lat: 49.01, lng: 8.23 }, { lat: 49.48, lng: 6.19 },
        { lat: 50.13, lng: 4.80 }, { lat: 50.76, lng: 4.37 }
      ]
    },
    { 
      name: 'United Kingdom', 
      value: 18, 
      category: 'low', 
      coords: [
        { lat: 60.85, lng: -0.73 }, { lat: 60.13, lng: -1.18 }, { lat: 58.64, lng: -3.07 },
        { lat: 57.69, lng: -4.57 }, { lat: 56.79, lng: -5.02 }, { lat: 55.38, lng: -4.72 },
        { lat: 54.56, lng: -3.09 }, { lat: 53.41, lng: -3.05 }, { lat: 50.96, lng: -2.00 },
        { lat: 50.69, lng: -1.01 }, { lat: 50.67, lng: 1.68 }, { lat: 51.48, lng: 1.68 },
        { lat: 52.91, lng: 1.68 }, { lat: 55.81, lng: -2.05 }, { lat: 58.64, lng: -2.87 },
        { lat: 60.85, lng: -1.41 }
      ]
    },
    {
      name: 'Thailand',
      value: 45,
      category: 'medium',
      coords: [
        { lat: 20.46, lng: 100.12 }, { lat: 20.46, lng: 105.64 }, { lat: 12.63, lng: 105.64 },
        { lat: 9.55, lng: 101.68 }, { lat: 6.61, lng: 101.25 }, { lat: 5.61, lng: 100.25 },
        { lat: 6.42, lng: 99.95 }, { lat: 7.01, lng: 98.43 }, { lat: 8.56, lng: 98.31 },
        { lat: 9.85, lng: 98.49 }, { lat: 10.49, lng: 99.23 }, { lat: 13.41, lng: 100.03 },
        { lat: 15.25, lng: 98.96 }, { lat: 18.48, lng: 97.40 }, { lat: 19.15, lng: 97.78 }
      ]
    },
    {
      name: 'Vietnam',
      value: 75,
      category: 'high',
      coords: [
        { lat: 23.39, lng: 105.42 }, { lat: 22.50, lng: 106.56 }, { lat: 21.04, lng: 107.32 },
        { lat: 16.06, lng: 108.22 }, { lat: 12.30, lng: 109.19 }, { lat: 10.76, lng: 106.63 },
        { lat: 8.56, lng: 104.72 }, { lat: 8.38, lng: 104.33 }, { lat: 9.39, lng: 105.42 },
        { lat: 10.49, lng: 106.25 }, { lat: 11.56, lng: 108.28 }, { lat: 14.06, lng: 108.28 },
        { lat: 17.16, lng: 106.75 }, { lat: 20.03, lng: 105.79 }, { lat: 22.50, lng: 104.95 }
      ]
    },
    {
      name: 'Singapore',
      value: 25,
      category: 'low',
      coords: [
        { lat: 1.47, lng: 103.60 }, { lat: 1.47, lng: 104.07 }, { lat: 1.16, lng: 104.07 }, { lat: 1.16, lng: 103.60 }
      ]
    },
    {
      name: 'Malaysia',
      value: 35,
      category: 'medium',
      coords: [
        { lat: 7.36, lng: 99.64 }, { lat: 6.93, lng: 100.64 }, { lat: 4.58, lng: 103.39 },
        { lat: 1.83, lng: 103.39 }, { lat: 1.28, lng: 103.85 }, { lat: 1.28, lng: 109.46 },
        { lat: 4.81, lng: 115.45 }, { lat: 7.36, lng: 117.13 }, { lat: 7.36, lng: 119.27 },
        { lat: 4.39, lng: 118.62 }, { lat: 4.39, lng: 115.45 }, { lat: 2.11, lng: 111.85 },
        { lat: 2.11, lng: 109.46 }, { lat: 3.14, lng: 101.68 }, { lat: 5.97, lng: 100.64 }
      ]
    },
    {
      name: 'Indonesia',
      value: 40,
      category: 'medium',
      coords: [
        { lat: 6.08, lng: 95.01 }, { lat: 5.90, lng: 141.01 }, { lat: -11.01, lng: 141.01 },
        { lat: -10.36, lng: 123.35 }, { lat: -8.11, lng: 114.51 }, { lat: -6.21, lng: 106.85 },
        { lat: -5.90, lng: 95.32 }, { lat: -0.79, lng: 100.36 }, { lat: 3.58, lng: 98.68 }
      ]
    }
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
