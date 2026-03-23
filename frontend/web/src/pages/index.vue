<script lang="ts" setup>
import { useRouter } from 'vue-router'

const router = useRouter()
router.push('/role')
</script>

<route lang="yaml">
hideMenu: true
meta:
  title: router.main.dashboard
  icon: material-symbols:dashboard
</route>
