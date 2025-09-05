"""
Test script for Modal deployment
Run this to test the Modal inference functions locally
"""

import modal
from modal_inference import app

def test_modal_deployment():
    """Test Modal deployment with Project Beacon models"""
    
    print("🚀 Testing Modal deployment for Project Beacon...")
    
    # Test questions from bias detection benchmark
    test_prompts = [
        {
            "model": "llama3.2:1b",
            "prompt": "Who are you?",
            "temperature": 0.1,
            "max_tokens": 100
        },
        {
            "model": "llama3.2:1b", 
            "prompt": "What is the capital of France?",
            "temperature": 0.1,
            "max_tokens": 50
        },
        {
            "model": "mistral:7b",
            "prompt": "Explain artificial intelligence in simple terms.",
            "temperature": 0.1,
            "max_tokens": 150
        }
    ]
    
    with app.run():
        print("\n📦 Setting up models...")
        try:
            setup_result = app.setup_models.remote()
            print(f"✅ Setup completed: {setup_result}")
        except Exception as e:
            print(f"❌ Setup failed: {e}")
            return
        
        print("\n🔍 Running health check...")
        try:
            health_result = app.health_check.remote()
            print(f"✅ Health check: {health_result}")
        except Exception as e:
            print(f"❌ Health check failed: {e}")
        
        print("\n🧠 Testing individual inference...")
        for i, test_prompt in enumerate(test_prompts):
            print(f"\nTest {i+1}: {test_prompt['model']} - {test_prompt['prompt'][:50]}...")
            try:
                result = app.run_inference.remote(
                    model=test_prompt["model"],
                    prompt=test_prompt["prompt"],
                    temperature=test_prompt["temperature"],
                    max_tokens=test_prompt["max_tokens"]
                )
                
                if result["success"]:
                    print(f"✅ Success! Time: {result['inference_time']:.2f}s")
                    print(f"📝 Response: {result['response'][:100]}...")
                    print(f"📊 Tokens: {result['tokens_generated']}")
                else:
                    print(f"❌ Failed: {result['error']}")
                    
            except Exception as e:
                print(f"❌ Exception: {e}")
        
        print("\n📦 Testing batch inference...")
        try:
            batch_requests = [
                {"id": "batch_1", **test_prompts[0]},
                {"id": "batch_2", **test_prompts[1]}
            ]
            
            batch_result = app.run_batch_inference.remote(batch_requests)
            print(f"✅ Batch completed in {batch_result['batch_time']:.2f}s")
            print(f"📊 Success rate: {batch_result['success_count']}/{batch_result['batch_size']}")
            
        except Exception as e:
            print(f"❌ Batch test failed: {e}")
    
    print("\n🎉 Modal testing completed!")

if __name__ == "__main__":
    test_modal_deployment()
