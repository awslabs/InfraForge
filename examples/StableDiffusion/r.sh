python3 generate_universal_test_config.py \
    --mode comparison \
    --models "stabilityai/stable-diffusion-2-1" "stabilityai/stable-diffusion-xl-base-1.0" \
    --instance-types "g4dn.xlarge" "g5.xlarge" "g6.xlarge" "g6e.xlarge" \
    --batch-sizes 1 4 \
    --inference-steps 15 25 50\
    --resolutions "1024x1024" \
    --precisions "float32" \
    --comparison-memory-request "12Gi" \
    --comparison-memory-limit "15Gi" \
    --comparison-timeout 1200 \
    --prompt "An astronaut riding a green horse" \
    --output "comparison_config.json"

#  --prompt "a photo of an astronaut riding a horse on mars" \
bash run_universal_tests.sh comparison_config.json
