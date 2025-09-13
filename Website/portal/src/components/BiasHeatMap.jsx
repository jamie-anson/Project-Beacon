import React, { useEffect, useRef, useState } from 'react';

export default function BiasHeatMap({ regionData = {}, className = '' }) {
  const mapRef = useRef(null);
  const [map, setMap] = useState(null);
  const [markers, setMarkers] = useState([]);
  const [isLoaded, setIsLoaded] = useState(false);

  // Region coordinates for markers
  const regionCoordinates = {
    'US': { lat: 39.8283, lng: -98.5795, name: 'United States' },
    'EU': { lat: 54.5260, lng: 15.2551, name: 'Europe' },
    'ASIA': { lat: 34.0479, lng: 100.6197, name: 'Asia Pacific' }
  };

  // Load Google Maps API
  useEffect(() => {
    if (window.google && window.google.maps) {
      setIsLoaded(true);
      return;
    }

    const script = document.createElement('script');
    script.src = `https://maps.googleapis.com/maps/api/js?key=${process.env.REACT_APP_GOOGLE_MAPS_API_KEY || 'YOUR_API_KEY'}&libraries=visualization`;
    script.async = true;
    script.defer = true;
    script.onload = () => setIsLoaded(true);
    script.onerror = () => console.error('Failed to load Google Maps API');
    
    document.head.appendChild(script);

    return () => {
      if (script.parentNode) {
        script.parentNode.removeChild(script);
      }
    };
  }, []);

  // Initialize map
  useEffect(() => {
    if (!isLoaded || !mapRef.current || map) return;

    const googleMap = new window.google.maps.Map(mapRef.current, {
      zoom: 2,
      center: { lat: 20, lng: 0 },
      mapTypeId: 'roadmap',
      styles: [
        {
          featureType: 'water',
          elementType: 'geometry',
          stylers: [{ color: '#e9e9e9' }, { lightness: 17 }]
        },
        {
          featureType: 'landscape',
          elementType: 'geometry',
          stylers: [{ color: '#f5f5f5' }, { lightness: 20 }]
        },
        {
          featureType: 'road.highway',
          elementType: 'geometry.fill',
          stylers: [{ color: '#ffffff' }, { lightness: 17 }]
        },
        {
          featureType: 'road.highway',
          elementType: 'geometry.stroke',
          stylers: [{ color: '#ffffff' }, { lightness: 29 }, { weight: 0.2 }]
        },
        {
          featureType: 'road.arterial',
          elementType: 'geometry',
          stylers: [{ color: '#ffffff' }, { lightness: 18 }]
        },
        {
          featureType: 'road.local',
          elementType: 'geometry',
          stylers: [{ color: '#ffffff' }, { lightness: 16 }]
        },
        {
          featureType: 'poi',
          elementType: 'geometry',
          stylers: [{ color: '#f5f5f5' }, { lightness: 21 }]
        },
        {
          featureType: 'poi.park',
          elementType: 'geometry',
          stylers: [{ color: '#dedede' }, { lightness: 21 }]
        },
        {
          elementType: 'labels.text.stroke',
          stylers: [{ visibility: 'on' }, { color: '#ffffff' }, { lightness: 16 }]
        },
        {
          elementType: 'labels.text.fill',
          stylers: [{ saturation: 36 }, { color: '#333333' }, { lightness: 40 }]
        },
        {
          elementType: 'labels.icon',
          stylers: [{ visibility: 'off' }]
        },
        {
          featureType: 'transit',
          elementType: 'geometry',
          stylers: [{ color: '#f2f2f2' }, { lightness: 19 }]
        },
        {
          featureType: 'administrative',
          elementType: 'geometry.fill',
          stylers: [{ color: '#fefefe' }, { lightness: 20 }]
        },
        {
          featureType: 'administrative',
          elementType: 'geometry.stroke',
          stylers: [{ color: '#fefefe' }, { lightness: 17 }, { weight: 1.2 }]
        }
      ]
    });

    setMap(googleMap);
  }, [isLoaded, map]);

  // Update markers when region data changes
  useEffect(() => {
    if (!map || !window.google) return;

    // Clear existing markers
    markers.forEach(marker => marker.setMap(null));

    const newMarkers = [];

    Object.entries(regionData).forEach(([region, data]) => {
      const coords = regionCoordinates[region];
      if (!coords || !data) return;

      const biasScore = data.scoring?.bias_score || 0;
      const censorshipScore = data.scoring?.censorship_score || 0;
      const overallRisk = Math.max(biasScore, censorshipScore);

      // Determine marker color based on risk level
      let markerColor = '#10B981'; // Green for low risk
      if (overallRisk >= 0.7) markerColor = '#EF4444'; // Red for high risk
      else if (overallRisk >= 0.4) markerColor = '#F59E0B'; // Yellow for medium risk

      // Create custom marker icon
      const markerIcon = {
        path: window.google.maps.SymbolPath.CIRCLE,
        fillColor: markerColor,
        fillOpacity: 0.8,
        strokeColor: '#ffffff',
        strokeWeight: 2,
        scale: Math.max(8, overallRisk * 20) // Size based on risk level
      };

      const marker = new window.google.maps.Marker({
        position: coords,
        map: map,
        icon: markerIcon,
        title: `${coords.name} - Risk: ${(overallRisk * 100).toFixed(1)}%`
      });

      // Create info window
      const infoWindow = new window.google.maps.InfoWindow({
        content: `
          <div style="padding: 8px; min-width: 200px;">
            <h3 style="margin: 0 0 8px 0; color: #1f2937; font-size: 14px; font-weight: 600;">
              ${coords.name}
            </h3>
            <div style="space-y: 4px;">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                <span style="color: #6b7280; font-size: 12px;">Bias Score:</span>
                <span style="color: #1f2937; font-size: 12px; font-weight: 500;">
                  ${(biasScore * 100).toFixed(1)}%
                </span>
              </div>
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                <span style="color: #6b7280; font-size: 12px;">Censorship:</span>
                <span style="color: #1f2937; font-size: 12px; font-weight: 500;">
                  ${(censorshipScore * 100).toFixed(1)}%
                </span>
              </div>
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                <span style="color: #6b7280; font-size: 12px;">Provider:</span>
                <span style="color: #1f2937; font-size: 12px; font-weight: 500;">
                  ${data.provider || 'Unknown'}
                </span>
              </div>
              <div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #e5e7eb;">
                <div style="color: #6b7280; font-size: 11px;">
                  Risk Level: 
                  <span style="color: ${markerColor}; font-weight: 600;">
                    ${overallRisk >= 0.7 ? 'HIGH' : overallRisk >= 0.4 ? 'MEDIUM' : 'LOW'}
                  </span>
                </div>
              </div>
            </div>
          </div>
        `
      });

      marker.addListener('click', () => {
        // Close other info windows
        newMarkers.forEach(m => {
          if (m.infoWindow) m.infoWindow.close();
        });
        infoWindow.open(map, marker);
      });

      marker.infoWindow = infoWindow;
      newMarkers.push(marker);
    });

    setMarkers(newMarkers);
  }, [map, regionData]);

  const Legend = () => (
    <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow-lg border p-3 z-10">
      <h4 className="text-sm font-medium text-slate-900 mb-2">Risk Levels</h4>
      <div className="space-y-1">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-red-500"></div>
          <span className="text-xs text-slate-600">High Risk (≥70%)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
          <span className="text-xs text-slate-600">Medium Risk (40-69%)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-green-500"></div>
          <span className="text-xs text-slate-600">Low Risk (<40%)</span>
        </div>
      </div>
      <div className="mt-2 pt-2 border-t text-xs text-slate-500">
        Click markers for details
      </div>
    </div>
  );

  if (!isLoaded) {
    return (
      <div className={`relative bg-slate-100 rounded-lg flex items-center justify-center ${className}`}>
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-beacon-600 mx-auto mb-2"></div>
          <div className="text-sm text-slate-600">Loading world map...</div>
        </div>
      </div>
    );
  }

  if (!process.env.REACT_APP_GOOGLE_MAPS_API_KEY && process.env.REACT_APP_GOOGLE_MAPS_API_KEY !== 'YOUR_API_KEY') {
    return (
      <div className={`relative bg-slate-100 rounded-lg flex items-center justify-center ${className}`}>
        <div className="text-center p-8">
          <div className="text-slate-500 mb-2">Google Maps API Key Required</div>
          <div className="text-sm text-slate-400">
            Set REACT_APP_GOOGLE_MAPS_API_KEY environment variable to enable map visualization
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      <div ref={mapRef} className="w-full h-full rounded-lg" />
      <Legend />
    </div>
  );
}
