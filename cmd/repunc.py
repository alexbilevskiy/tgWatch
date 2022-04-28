import os
import torch
import yaml
from torch import package

models_yaml = 'latest_silero_models.yml'
if not os.path.exists(models_yaml):
    torch.hub.download_url_to_file('https://raw.githubusercontent.com/snakers4/silero-models/master/models.yml', models_yaml, progress=False)

with open(models_yaml, 'r') as yaml_file:
    models = yaml.load(yaml_file, Loader=yaml.SafeLoader)
model_conf = models.get('te_models').get('latest')

model_url = model_conf.get('package')

model_dir = "downloaded_model"
os.makedirs(model_dir, exist_ok=True)
model_path = os.path.join(model_dir, os.path.basename(model_url))

if not os.path.isfile(model_path):
    torch.hub.download_url_to_file(model_url, model_path, progress=True)

imp = package.PackageImporter(model_path)
model = imp.load_pickle("te_model", "model")
example_texts = model.examples


def apply_te(text, lan='en'):
    return model.enhance_text(text, lan)


input_text = input()
print(apply_te(input_text, lan='ru'))